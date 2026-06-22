package agent

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
)

// TestAgentReportQueuesPreparedBatch проверяет границу Agent -> ReportingService:
// runtime ставит подготовленный сервисом batch в очередь отправки.
func TestAgentReportQueuesPreparedBatch(t *testing.T) {
	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	service := NewReportingService(repo, newNoopSender(), zap.NewNop())
	a, err := New(service, config.AgentConfig{
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           1,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	service.pollSinceReport.Store(3)
	a.report()

	var task reportTask
	select {
	case task = <-a.recvQueue:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for report task")
	}

	assertMetricValue(t, task.metrics, "Alloc", 5.11)
	assertMetricDelta(t, task.metrics, "PollCount", 3)

	if got := service.pollSinceReport.Load(); got != 0 {
		t.Fatalf("pollSinceReport = %d, want 0", got)
	}
}

// TestAgentReportRestoresPollCountWhenQueueIsFull проверяет,
// что runtime возвращает сервису счётчик непринятой очередью задачи.
func TestAgentReportRestoresPollCountWhenQueueIsFull(t *testing.T) {
	service := NewReportingService(
		repository.NewMemStorage(),
		newNoopSender(),
		zap.NewNop(),
	)
	a, err := New(service, config.AgentConfig{
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           1,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	service.pollSinceReport.Store(1)
	a.report()

	service.pollSinceReport.Store(2)
	a.report()

	if got, want := service.pollSinceReport.Load(), int64(2); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}
