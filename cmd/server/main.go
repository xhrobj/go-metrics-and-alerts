package main

import (
	"context"
	"crypto/rsa"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/buildinfo"
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/migrations"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const shutdownTimeout = time.Second * 30

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	if err := buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit); err != nil {
		log.Fatal(err)
	}

	if err := printBanner(os.Stdout); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer stop()

	if err := run(ctx); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.GetServerConfig()
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}

	privateKey, err := loadPrivateKey(cfg.CryptoKey)
	if err != nil {
		return err
	}

	trustedSubnet, err := parseTrustedSubnet(cfg.TrustedSubnet)
	if err != nil {
		return err
	}

	db, err := openDatabase(cfg, lg)
	if err != nil {
		return err
	}
	if db != nil {
		defer func() {
			_ = db.Close()
		}()
	}

	repo, memRepo := newMetricsStorage(db)
	svc := service.NewMetricsService(repo)

	h := handler.New(svc, db)

	cleanupAudit, err := setupAudit(cfg, h, lg)
	if err != nil {
		return err
	}
	defer cleanupAudit()

	r := router.New(h, lg, router.Options{
		HashKey:       cfg.Key,
		PrivateKey:    privateKey,
		TrustedSubnet: trustedSubnet,
	})

	listener, err := net.Listen("tcp", cfg.ServerAddr)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
	}()

	shutdownPersistence, err := setupFilePersistence(cfg, svc, memRepo, lg)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	lg.Info("running server",
		zap.String("address", cfg.ServerAddr),
		zap.String("fileStoragePath", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
		zap.Int("storeIntervalInSec", cfg.StoreIntervalInSec),
	)

	serveErr := serve(ctx, srv, listener, lg)
	persistenceErr := shutdownPersistence()

	if serveErr == nil && persistenceErr == nil {
		lg.Info("server stopped")
	}

	return errors.Join(serveErr, persistenceErr)
}

func serve(ctx context.Context, srv *http.Server, listener net.Listener, lg *zap.Logger) error {
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
		lg.Info("shutdown signal received")
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

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil
	}

	privateKey, err := encryption.LoadPrivateKey(path)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	return privateKey, nil
}

func parseTrustedSubnet(value string) (*net.IPNet, error) {
	if value == "" {
		return nil, nil
	}

	_, subnet, err := net.ParseCIDR(value)
	if err != nil {
		return nil, fmt.Errorf("parse trusted subnet %q: %w", value, err)
	}

	return subnet, nil
}

func openDatabase(cfg config.ServerConfig, lg *zap.Logger) (*sql.DB, error) {
	if cfg.DatabaseDSN == "" {
		lg.Debug("database disabled")
		return nil, nil
	}

	db, err := sql.Open("pgx", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	lg.Debug("database connected")

	if err := migrations.RunMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func newMetricsStorage(db *sql.DB) (service.MetricsStorage, *repository.MemStorage) {
	if db != nil {
		return repository.NewPostgresStorage(db), nil
	}

	memRepo := repository.NewMemStorage()

	return memRepo, memRepo
}

func setupFilePersistence(
	cfg config.ServerConfig,
	svc *service.MetricsService,
	memRepo *repository.MemStorage,
	lg *zap.Logger,
) (func() error, error) {
	shutdown := func() error { return nil }

	if memRepo == nil || cfg.FileStoragePath == "" {
		return shutdown, nil
	}

	store := repository.NewFileStore(cfg.FileStoragePath)

	if cfg.Restore {
		if err := store.Load(memRepo); err != nil {
			return nil, err
		}
	}

	var stopPeriodicSave func()

	switch {
	case cfg.StoreIntervalInSec == 0:
		svc.EnableSyncSave(store)

	case cfg.StoreIntervalInSec > 0:
		stopPeriodicSave = startPeriodicSave(
			store,
			memRepo,
			time.Duration(cfg.StoreIntervalInSec)*time.Second,
			cfg.FileStoragePath,
			lg,
		)

	default:
		return nil, fmt.Errorf(
			"store interval in seconds must be >= 0, got %d",
			cfg.StoreIntervalInSec,
		)
	}

	shutdown = func() error {
		if stopPeriodicSave != nil {
			stopPeriodicSave()
		}

		if err := store.Save(context.Background(), memRepo); err != nil {
			return fmt.Errorf("save metrics on shutdown: %w", err)
		}

		return nil
	}

	return shutdown, nil
}

func startPeriodicSave(
	store *repository.FileStore,
	memRepo *repository.MemStorage,
	interval time.Duration,
	path string,
	lg *zap.Logger,
) func() {
	stopCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		defer close(doneCh)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := store.Save(context.Background(), memRepo); err != nil {
					lg.Error(
						"failed to save metrics to file",
						zap.String("path", path),
						zap.Error(err),
					)
				}

			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		close(stopCh)
		<-doneCh
	}
}

func setupAudit(
	cfg config.ServerConfig,
	h *handler.Handler,
	lg *zap.Logger,
) (func(), error) {
	noop := func() {
		// Освобождать ресурсы не требуется
	}

	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return noop, nil
	}

	cleanup := noop
	auditDispatcher := audit.NewAuditor()

	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, err
		}

		cleanup = func() {
			if err := fileObserver.Close(); err != nil {
				lg.Error("failed to close audit file", zap.Error(err))
			}
		}

		auditDispatcher.Subscribe(fileObserver)
	}

	if cfg.AuditURL != "" {
		auditDispatcher.Subscribe(audit.NewRemoteObserver(cfg.AuditURL))
	}

	h.EnableAudit(auditDispatcher, lg)

	return cleanup, nil
}

func printBanner(w io.Writer) error {
	const banner = `
   _____          __         .__                _________
  /     \   _____/  |________|__| ____   ______/   _____/ ______________  __ ___________
 /  \ /  \_/ __ \   __\_  __ \  |/ ___\ /  ___/\_____  \_/ __ \_  __ \  \/ // __ \_  __ \
/    Y    \  ___/|  |  |  | \/  \  \___ \___ \ /        \  ___/|  | \/\   /\  ___/|  | \/
\____|__  /\___  >__|  |__|  |__|\___  >____  >_______  /\___  >__|    \_/  \___  >__|
        \/     \/                    \/     \/        \/     \/                 \/

`
	_, err := io.WriteString(w, banner)

	return err
}
