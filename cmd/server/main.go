package main

import (
	"log"
	"net/http"

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

	/* repo.UpdateGauge("Alloc", 123.45)
	repo.UpdateCounter("PollCount", 7)

	fileStorage := repository.NewFileStorage("metrics-db.json")
	if err := fileStorage.Save(repo); err != nil {
		return err
	} */

	h := handler.New(repo)
	r := router.New(h, zapLogger)

	zapLogger.Info("running server",
		zap.String("address", cfg.ServerAddr),
	)

	return http.ListenAndServe(cfg.ServerAddr, r)
}
