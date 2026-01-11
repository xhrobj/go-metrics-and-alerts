package agent

import (
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// poll() увеличивает PollCount и выставляет RandomValue
func TestAgent_Poll_UpdatesMetrics(t *testing.T) {
	repo := repository.NewMemStorage()
	a := New(repo, "http://example.com:8080", 2, 10)

	a.poll()

	gauges, counters := repo.Snapshot()

	// PollCount
	if got := counters["PollCount"]; got != 1 {
		t.Fatalf("expected PollCount=1, got %d", got)
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
