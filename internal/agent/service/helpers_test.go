package service

import (
	"context"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

type senderFunc func(context.Context, []model.Metrics) error

func (f senderFunc) Send(ctx context.Context, metrics []model.Metrics) error {
	return f(ctx, metrics)
}

func newNoopSender() MetricsSender {
	return senderFunc(func(context.Context, []model.Metrics) error {
		return nil
	})
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

func assertMetricDelta(t *testing.T, metrics []model.Metrics, name string, want int64) {
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
