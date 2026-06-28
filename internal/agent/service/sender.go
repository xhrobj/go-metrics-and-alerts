package service

import (
	"context"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

// MetricsSender отправляет batch метрик на Сервер.
type MetricsSender interface {
	Send(context.Context, []model.Metrics) error
}
