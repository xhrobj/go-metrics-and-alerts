package agent

import (
	"context"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// poll() выставляет RandomValue, добавляет runtime-метрики
func TestAgent_Poll_UpdatesMetrics(t *testing.T) {
	repo := repository.NewMemStorage()
	a, _ := New(repo, "example.com:8080", 2, 10, 5, "secret-key")

	a.poll()

	gauges, _, _ := repo.Snapshot(context.Background())

	// RandomValue должен существовать
	if _, ok := gauges["RandomValue"]; !ok {
		t.Fatalf("expected RandomValue gauge to be set")
	}

	// хотя бы одна runtime-метрика
	if _, ok := gauges["Alloc"]; !ok {
		t.Fatalf("expected runtime metric Alloc to be set")
	}
}
