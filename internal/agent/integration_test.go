package agent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/service"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
)

type senderFunc func(context.Context, []model.Metrics) error

func (f senderFunc) Send(ctx context.Context, metrics []model.Metrics) error {
	return f(ctx, metrics)
}

// TestRunDrainsQueueAndSendsFinalSnapshot проверяет исходный сквозной сценарий:
// при shutdown Агент дожидается активной отправки, опустошает очередь и
// отправляет через настоящий ReportingService финальный снимок.
func TestRunDrainsQueueAndSendsFinalSnapshot(t *testing.T) {
	firstSendStarted := make(chan struct{})
	allowFirstSendToFinish := make(chan struct{})
	sentMetrics := make(chan []model.Metrics, 2)

	var sendCount atomic.Int32
	sender := senderFunc(func(_ context.Context, metrics []model.Metrics) error {
		sentMetrics <- metrics
		if sendCount.Add(1) == 1 {
			close(firstSendStarted)
			<-allowFirstSendToFinish
		}
		return nil
	})

	t.Cleanup(func() {
		select {
		case <-allowFirstSendToFinish:
		default:
			close(allowFirstSendToFinish)
		}
	})

	cfg := config.AgentConfig{
		PollIntervalInSec:   3600,
		ReportIntervalInSec: 3600,
		RateLimit:           1,
	}

	repo := repository.NewMemStorage()
	reportingService := service.New(repo, sender, zap.NewNop())

	for range 42 {
		reportingService.PollRuntime()
	}
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	a, err := New(reportingService, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	a.report()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- a.Run(ctx)
	}()

	select {
	case <-firstSendStarted:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for active send")
	}

	for range 7 {
		reportingService.PollRuntime()
	}
	if err := repo.UpdateGauge(context.Background(), "Alloc", 6.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	cancel()

	select {
	case err := <-runErrCh:
		t.Fatalf("Run() returned before active send completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(allowFirstSendToFinish)

	var gotBatches [][]model.Metrics
	for range 2 {
		select {
		case metrics := <-sentMetrics:
			gotBatches = append(gotBatches, metrics)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for metrics batch")
		}
	}

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run() to stop")
	}

	if got, want := sendCount.Load(), int32(2); got != want {
		t.Fatalf("send count = %d, want %d", got, want)
	}

	assertMetricValue(t, gotBatches[0], "Alloc", 5.11)
	assertMetricDelta(t, gotBatches[0], "PollCount", 42)
	assertMetricValue(t, gotBatches[1], "Alloc", 6.11)
	assertMetricDelta(t, gotBatches[1], "PollCount", 7)
}

func assertMetricValue(
	t *testing.T,
	metrics []model.Metrics,
	name string,
	want float64,
) {
	t.Helper()

	for _, metric := range metrics {
		if metric.ID == name && metric.MType == model.Gauge {
			if metric.Value == nil {
				t.Fatalf("metric %q value = nil, want %v", name, want)
			}
			if got := *metric.Value; got != want {
				t.Fatalf("metric %q value = %v, want %v", name, got, want)
			}
			return
		}
	}

	t.Fatalf("metric %q not found", name)
}

func assertMetricDelta(
	t *testing.T,
	metrics []model.Metrics,
	name string,
	want int64,
) {
	t.Helper()

	for _, metric := range metrics {
		if metric.ID == name && metric.MType == model.Counter {
			if metric.Delta == nil {
				t.Fatalf("metric %q delta = nil, want %d", name, want)
			}
			if got := *metric.Delta; got != want {
				t.Fatalf("metric %q delta = %d, want %d", name, got, want)
			}
			return
		}
	}

	t.Fatalf("metric %q not found", name)
}
