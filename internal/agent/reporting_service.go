package agent

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

type reportTask struct {
	metrics   []model.Metrics
	pollCount int64
}

// ReportingService реализует прикладные сценарии Агента:
// сбор метрик, формирование batch-отчётов и их отправку.
type ReportingService struct {
	repo   AgentStorage
	sender MetricsSender
	log    *zap.Logger

	// pollSinceReport - количество вызовов pollRuntime() с момента последней
	// успешной отправки отчёта. Используется для формирования метрики PollCount.
	pollSinceReport atomic.Int64
}

// NewReportingService создаёт сервис отчётности Агента.
func NewReportingService(
	repo AgentStorage,
	sender MetricsSender,
	log *zap.Logger,
) *ReportingService {
	if log == nil {
		log = zap.NewNop()
	}

	return &ReportingService{
		repo:   repo,
		sender: sender,
		log:    log,
	}
}
