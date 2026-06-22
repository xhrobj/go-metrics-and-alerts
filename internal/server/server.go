package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/service"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/router"
	"go.uber.org/zap"
)

const shutdownTimeout = time.Second * 30

// Server управляет lifecycle Сервера и принадлежащими ему ресурсами.
type Server struct {
	cfg config.ServerConfig
	log *zap.Logger
	db  *sql.DB

	httpServer   *http.Server
	httpListener net.Listener

	shutdownPersistence func() error
	persistenceOnce     sync.Once
	persistenceErr      error
	cleanupAudit        func()
	closeOnce           sync.Once
}

// New создаёт Сервер, настраивает зависимости и открывает HTTP-listener.
func New(cfg config.ServerConfig, log *zap.Logger) (_ *Server, err error) {
	if log == nil {
		log = zap.NewNop()
	}

	srv := &Server{
		cfg: cfg,
		log: log,
		shutdownPersistence: func() error {
			// 4Sonar: Persistence может быть отключен, завершать нечего
			return nil
		},
		cleanupAudit: func() {
			// 4Sonar: Аудит может быть отключен, освобождать ресурсы не требуется
		},
	}

	defer func() {
		if err != nil {
			srv.Close()
		}
	}()

	privateKey, err := loadPrivateKey(cfg.CryptoKey)
	if err != nil {
		return nil, err
	}

	trustedSubnet, err := parseTrustedSubnet(cfg.TrustedSubnet)
	if err != nil {
		return nil, err
	}

	srv.db, err = openDatabase(cfg, log)
	if err != nil {
		return nil, err
	}

	repo, memRepo := newMetricsStorage(srv.db)
	metricsService := service.NewMetricsService(repo)

	h := handler.New(metricsService, srv.db)

	srv.cleanupAudit, err = setupAudit(cfg, h, log)
	if err != nil {
		return nil, err
	}

	httpHandler := router.New(h, log, router.Options{
		HashKey:       cfg.Key,
		PrivateKey:    privateKey,
		TrustedSubnet: trustedSubnet,
	})

	srv.httpListener, err = net.Listen("tcp", cfg.ServerAddr)
	if err != nil {
		return nil, err
	}

	srv.shutdownPersistence, err = setupFilePersistence(cfg, metricsService, memRepo, log)
	if err != nil {
		return nil, err
	}

	srv.httpServer = &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: httpHandler,
	}

	return srv, nil
}

// Run запускает HTTP-Сервер и выполняет штатное завершение после отмены контекста.
func (s *Server) Run(ctx context.Context) error {
	s.log.Info("running server",
		zap.String("address", s.cfg.ServerAddr),
		zap.String("fileStoragePath", s.cfg.FileStoragePath),
		zap.Bool("restore", s.cfg.Restore),
		zap.Int("storeIntervalInSec", s.cfg.StoreIntervalInSec),
	)

	serveErr := serveHTTP(ctx, s.httpServer, s.httpListener, s.log)
	persistenceErr := s.stopPersistence()

	if serveErr == nil && persistenceErr == nil {
		s.log.Info("server stopped")
	}

	return errors.Join(serveErr, persistenceErr)
}

// Close освобождает ресурсы Сервера. Повторный вызов безопасен.
func (s *Server) Close() {
	if s == nil {
		return
	}

	s.closeOnce.Do(func() {
		_ = s.stopPersistence()

		if s.httpListener != nil {
			_ = s.httpListener.Close()
		}

		if s.cleanupAudit != nil {
			s.cleanupAudit()
		}

		if s.db != nil {
			_ = s.db.Close()
		}
	})
}

func (s *Server) stopPersistence() error {
	s.persistenceOnce.Do(func() {
		s.persistenceErr = s.shutdownPersistence()
	})

	return s.persistenceErr
}

func serveHTTP(ctx context.Context, srv *http.Server, httpListener net.Listener, log *zap.Logger) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Serve(httpListener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err

	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		shutdownTimeout,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()

		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
