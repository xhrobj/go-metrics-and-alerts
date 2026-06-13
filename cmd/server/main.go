package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/buildinfo"
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/migrations"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

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

	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.GetServerConfig()
	if err != nil {
		return err
	}

	lg, err := logger.New()
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

	if err := setupFilePersistence(cfg, svc, memRepo, lg); err != nil {
		return err
	}

	h := handler.New(svc, db)

	cleanupAudit, err := setupAudit(cfg, h, lg)
	if err != nil {
		return err
	}
	defer cleanupAudit()

	r := router.New(h, lg, router.Options{
		HashKey: cfg.Key,
	})

	lg.Info("running server",
		zap.String("address", cfg.ServerAddr),
		zap.String("fileStoragePath", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
		zap.Int("storeIntervalInSec", cfg.StoreIntervalInSec),
	)

	return http.ListenAndServe(cfg.ServerAddr, r)
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
) error {
	if memRepo == nil {
		return nil
	}

	store := repository.NewFileStore(cfg.FileStoragePath)

	if cfg.Restore {
		if err := store.Load(memRepo); err != nil {
			return err
		}
	}

	switch {
	case cfg.StoreIntervalInSec == 0:
		svc.EnableSyncSave(store)

	case cfg.StoreIntervalInSec > 0:
		go func() {
			ticker := time.NewTicker(
				time.Duration(cfg.StoreIntervalInSec) * time.Second,
			)
			defer ticker.Stop()

			for range ticker.C {
				if err := store.Save(context.Background(), memRepo); err != nil {
					lg.Error(
						"failed to save metrics to file",
						zap.String("path", cfg.FileStoragePath),
						zap.Error(err),
					)
				}
			}
		}()

	default:
		return fmt.Errorf(
			"store interval in seconds must be >= 0, got %d",
			cfg.StoreIntervalInSec,
		)
	}

	return nil
}

func setupAudit(
	cfg config.ServerConfig,
	h *handler.Handler,
	lg *zap.Logger,
) (func(), error) {
	cleanup := func() {}

	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return cleanup, nil
	}

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
