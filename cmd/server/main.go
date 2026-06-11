package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"database/sql"

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
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if err := buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit); err != nil {
		log.Fatal(err)
	}

	if err := run(); err != nil {
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

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		db, err = sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			return err
		}
		defer func() {
			_ = db.Close()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			return err
		}

		lg.Debug("database connected")

		if err := migrations.RunMigrations(db); err != nil {
			return err
		}
	} else {
		lg.Debug("database disabled")
	}

	var repo service.MetricsStorage
	var memRepo *repository.MemStorage

	if db != nil {
		repo = repository.NewPostgresStorage(db)
	} else {
		memRepo = repository.NewMemStorage()
		repo = memRepo
	}

	svc := service.NewMetricsService(repo)

	if memRepo != nil {
		store := repository.NewFileStore(cfg.FileStoragePath)

		if cfg.Restore {
			if err := store.Load(memRepo); err != nil {
				return err
			}
		}

		if cfg.StoreIntervalInSec == 0 {
			svc.EnableSyncSave(store)
		} else if cfg.StoreIntervalInSec > 0 {
			go func() {
				ticker := time.NewTicker(time.Duration(cfg.StoreIntervalInSec) * time.Second)
				defer ticker.Stop()

				for range ticker.C {
					if err := store.Save(memRepo); err != nil {
						lg.Error("failed to save metrics to file",
							zap.String("path", cfg.FileStoragePath),
							zap.Error(err),
						)
					}
				}
			}()
		} else {
			return fmt.Errorf("store interval in seconds must be >= 0, got %d", cfg.StoreIntervalInSec)
		}
	}

	h := handler.New(svc, db)

	if cfg.AuditFile != "" || cfg.AuditURL != "" {
		auditDispatcher := audit.NewAuditor()

		if cfg.AuditFile != "" {
			fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
			if err != nil {
				return err
			}

			defer func() {
				if err := fileObserver.Close(); err != nil {
					lg.Error("failed to close audit file", zap.Error(err))
				}
			}()

			auditDispatcher.Subscribe(fileObserver)
		}

		if cfg.AuditURL != "" {
			auditDispatcher.Subscribe(audit.NewRemoteObserver(cfg.AuditURL))
		}

		h.EnableAudit(auditDispatcher, lg)
	}

	r := router.New(h, lg, cfg.Key)

	lg.Info("running server",
		zap.String("address", cfg.ServerAddr),
		zap.String("fileStoragePath", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
		zap.Int("storeIntervalInSec", cfg.StoreIntervalInSec),
	)

	return http.ListenAndServe(cfg.ServerAddr, r)
}
