package main

import (
	"log"
	"net/http"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
	"go.uber.org/zap"
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

	zapLogger, err := logger.New()
	if err != nil {
		return err
	}

	repo := repository.NewMemStorage()
	fileStorage := repository.NewFileStorage(cfg.FileStoragePath)

	if cfg.Restore {
		if err := fileStorage.Load(repo); err != nil {
			return err
		}
	}

	if cfg.StoreIntervalInSec > 0 {
		go func() {
			ticker := time.NewTicker(time.Duration(cfg.StoreIntervalInSec) * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				if err := fileStorage.Save(repo); err != nil {
					zapLogger.Error("failed to save metrics to file",
						zap.String("path", cfg.FileStoragePath),
						zap.Error(err),
					)
				}
			}
		}()
	}

	h := handler.New(repo)
	r := router.New(h, zapLogger)

	zapLogger.Info("running server",
		zap.String("address", cfg.ServerAddr),
	)

	return http.ListenAndServe(cfg.ServerAddr, r)
}
