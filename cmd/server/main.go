package main

import (
	"log"
	"net/http"
	"time"

	"database/sql"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
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

	if cfg.DatabaseDSN != "" {
		db, err := sql.Open("pgx", cfg.DatabaseDSN)
		if err != nil {
			return err
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			return err
		}

		lg.Info("(-_-) database connected")
	}

	repo := repository.NewMemStorage()
	store := repository.NewFileStore(cfg.FileStoragePath)

	if cfg.Restore {
		if err := store.Load(repo); err != nil {
			return err
		}
	}

	svc := service.NewMetricsService(repo)

	if cfg.StoreIntervalInSec == 0 {
		svc.EnableSyncSave(store)
	} else if cfg.StoreIntervalInSec > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreIntervalInSec) * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := store.Save(repo); err != nil {
					lg.Error("failed to save metrics to file",
						zap.String("path", cfg.FileStoragePath),
						zap.Error(err),
					)
				}
			}
		}()
	}

	h := handler.New(svc)
	r := router.New(h, lg)

	lg.Info("running server",
		zap.String("address", cfg.ServerAddr),
		zap.String("fileStoragePath", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
		zap.Int("storeIntervalInSec", cfg.StoreIntervalInSec),
	)

	return http.ListenAndServe(cfg.ServerAddr, r)
}
