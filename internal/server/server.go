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

	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/service"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/router"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const shutdownTimeout = time.Second * 30

// Server управляет lifecycle Сервера и принадлежащими ему ресурсами.
type Server struct {
	cfg config.ServerConfig
	log *zap.Logger

	db *sql.DB

	httpServer   *http.Server
	httpListener net.Listener

	grpcServer   *grpc.Server
	grpcListener net.Listener

	shutdownPersistence func() error
	persistenceOnce     sync.Once
	persistenceErr      error

	auditor      *audit.Auditor
	cleanupAudit func()

	closeOnce sync.Once
}

// New создаёт Сервер, настраивает зависимости и открывает HTTP/gRPC-listener'ы.
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

	srv.auditor, srv.cleanupAudit, err = setupAudit(cfg, log)
	if err != nil {
		return nil, err
	}

	h := handler.New(metricsService, srv.db)

	if srv.auditor != nil {
		h.EnableAudit(srv.auditor, log)
	}

	httpHandler := router.New(h, log, router.Options{
		HashKey:       cfg.Key,
		PrivateKey:    privateKey,
		TrustedSubnet: trustedSubnet,
	})

	srv.httpServer = &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: httpHandler,
	}

	srv.grpcServer, err = newGRPCServer(
		metricsService,
		srv.auditor,
		trustedSubnet,
		cfg.GRPCTLSCert,
		cfg.GRPCTLSKey,
		log,
	)
	if err != nil {
		return nil, err
	}

	srv.httpListener, err = net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return nil, fmt.Errorf("listen HTTP: %w", err)
	}

	srv.grpcListener, err = net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		return nil, fmt.Errorf("listen gRPC: %w", err)
	}

	srv.shutdownPersistence, err = setupFilePersistence(
		cfg,
		metricsService,
		memRepo,
		log,
	)
	if err != nil {
		return nil, err
	}

	return srv, nil
}

// Run запускает gRPC/HTTP-серверы и выполняет штатное завершение после отмены контекста.
func (s *Server) Run(ctx context.Context) error {
	s.log.Info("(^_^) running Server",
		zap.String("httpAddress", s.cfg.HTTPAddr),
		zap.String("grpcAddress", s.cfg.GRPCAddr),
		zap.Bool("grpcTLS", s.cfg.GRPCTLSCert != ""),
		zap.String("fileStoragePath", s.cfg.FileStoragePath),
		zap.Bool("restore", s.cfg.Restore),
		zap.Int("storeIntervalInSec", s.cfg.StoreIntervalInSec),
	)

	serveErr := serveTransports(
		ctx,
		s.httpServer,
		s.httpListener,
		s.grpcServer,
		s.grpcListener,
		s.log,
	)

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

		if s.httpServer != nil {
			_ = s.httpServer.Close()
		}

		if s.httpListener != nil {
			_ = s.httpListener.Close()
		}

		if s.grpcServer != nil {
			s.grpcServer.Stop()
		}

		if s.grpcListener != nil {
			_ = s.grpcListener.Close()
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

func serveTransports(
	ctx context.Context,
	httpServer *http.Server,
	httpListener net.Listener,
	grpcServer *grpc.Server,
	grpcListener net.Listener,
	log *zap.Logger,
) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		err := serveHTTP(runCtx, httpServer, httpListener, log)
		if err != nil {
			err = fmt.Errorf("serve HTTP: %w", err)
		}
		errCh <- err
	}()

	go func() {
		err := serveGRPC(runCtx, grpcServer, grpcListener, log)
		if err != nil {
			err = fmt.Errorf("serve gRPC: %w", err)
		}
		errCh <- err
	}()

	firstErr := <-errCh
	cancel()
	secondErr := <-errCh

	return errors.Join(firstErr, secondErr)
}

func serveHTTP(ctx context.Context, srv *http.Server, listener net.Listener, log *zap.Logger) error {
	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err

	case <-ctx.Done():
		log.Info("shutdown signal received", zap.String("transport", "HTTP"))
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
