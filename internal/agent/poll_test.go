package agent

import (
	"context"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// pollRuntime() выставляет RandomValue, добавляет runtime-метрики.
func TestAgent_PollRuntime_UpdatesMetrics(t *testing.T) {
	repo := repository.NewMemStorage()
	a, err := New(repo, "example.com:8080", 2, 10, 5, "secret-key")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	a.pollRuntime()

	gauges, _, err := repo.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	// RandomValue должен существовать
	if _, ok := gauges["RandomValue"]; !ok {
		t.Fatalf("expected RandomValue gauge to be set")
	}

	// хотя бы одна runtime-метрика
	if _, ok := gauges["Alloc"]; !ok {
		t.Fatalf("expected runtime metric Alloc to be set")
	}
}

// pollSystem() сохраняет в хранилище системные метрики.
func TestAgent_PollSystem_StoresSystemMetrics(t *testing.T) {
	repo := repository.NewMemStorage()

	a, err := New(repo, "localhost:8080", 2, 10, 5, "")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	a.pollSystem()

	gauges, _, err := repo.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}

	if _, ok := gauges["TotalMemory"]; !ok {
		t.Fatalf("expected TotalMemory metric to be present")
	}

	if _, ok := gauges["FreeMemory"]; !ok {
		t.Fatalf("expected FreeMemory metric to be present")
	}

	if _, ok := gauges["CPUutilization1"]; !ok {
		t.Fatalf("expected CPUutilization1 metric to be present")
	}
}
