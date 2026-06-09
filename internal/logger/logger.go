package logger

import "go.uber.org/zap"

// New создаёт production-логгер приложения.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	return cfg.Build()
}
