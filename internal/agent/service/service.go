// Package service реализует сценарии работы Агента:
// сбор метрик, формирование отчётов и их отправку.
package service

import (
	"context"
	"sync/atomic"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"go.uber.org/zap"
)

// AgentStorage описывает хранилище метрик, используемое ReportingService.
type AgentStorage interface {
	UpdateGauge(context.Context, string, float64) error
	Snapshot(context.Context) (map[string]float64, map[string]int64, error)
}

// Report содержит подготовленный batch метрик и соответствующее ему значение PollCount.
type Report struct {
	Metrics   []model.Metrics
	PollCount int64
}

// ReportingService реализует сценарии работы Агента:
// сбор метрик, формирование batch-отчётов и их отправку.
type ReportingService struct {
	repo   AgentStorage
	sender MetricsSender
	log    *zap.Logger

	// pollSinceReport - количество вызовов PollRuntime() с момента последней
	// успешной отправки отчёта. Используется для формирования метрики PollCount.
	pollSinceReport atomic.Int64
}

// New создаёт сервис отчётности Агента.
func New(repo AgentStorage, sender MetricsSender, log *zap.Logger) *ReportingService {
	if log == nil {
		log = zap.NewNop()
	}

	return &ReportingService{
		repo:   repo,
		sender: sender,
		log:    log,
	}
}
