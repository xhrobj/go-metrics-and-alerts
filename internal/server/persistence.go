package server

import (
	"context"
	"fmt"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/service"
	"go.uber.org/zap"
)

func setupFilePersistence(
	cfg config.ServerConfig,
	metricsService *service.MetricsService,
	memRepo *repository.MemStorage,
	log *zap.Logger,
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
		metricsService.EnableSyncSave(store)

	case cfg.StoreIntervalInSec > 0:
		stopPeriodicSave = startPeriodicSave(
			store,
			memRepo,
			time.Duration(cfg.StoreIntervalInSec)*time.Second,
			cfg.FileStoragePath,
			log,
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
	log *zap.Logger,
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
					log.Error(
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
