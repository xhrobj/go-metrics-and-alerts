package agent

import (
	"context"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
)

// pollRuntime() выставляет RandomValue и добавляет runtime-метрики.
func TestReportingServicePollRuntimeUpdatesMetrics(t *testing.T) {
	repo := repository.NewMemStorage()
	service := NewReportingService(repo, newNoopSender(), zap.NewNop())

	service.pollRuntime()

	gauges, _, err := repo.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}

	if _, ok := gauges["RandomValue"]; !ok {
		t.Fatal("RandomValue not found")
	}
	if _, ok := gauges["Alloc"]; !ok {
		t.Fatal("Alloc not found")
	}
	if got, want := service.pollSinceReport.Load(), int64(1); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}

// pollSystem() сохраняет в хранилище системные метрики.
func TestReportingServicePollSystemStoresSystemMetrics(t *testing.T) {
	repo := repository.NewMemStorage()
	service := NewReportingService(repo, newNoopSender(), zap.NewNop())

	service.pollSystem()

	gauges, _, err := repo.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v, want nil", err)
	}

	if _, ok := gauges["TotalMemory"]; !ok {
		t.Fatal("TotalMemory not found")
	}
	if _, ok := gauges["FreeMemory"]; !ok {
		t.Fatal("FreeMemory not found")
	}
	if _, ok := gauges["CPUutilization1"]; !ok {
		t.Fatal("CPUutilization1 not found")
	}
}
