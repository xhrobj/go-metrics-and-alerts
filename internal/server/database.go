package server

import (
	"context"
	"database/sql"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/migrations"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/service"
	"go.uber.org/zap"

	// Регистрирует драйвер pgx в database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
)

func openDatabase(cfg config.ServerConfig, log *zap.Logger) (*sql.DB, error) {
	if cfg.DatabaseDSN == "" {
		log.Debug("database disabled")
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

	log.Debug("database connected")

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
