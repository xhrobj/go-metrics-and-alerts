package service_test

import (
	"context"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
)

func TestMetricsService_UpdateAndGetGauge(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	err := svc.UpdateGauge(ctx, "Alloc", 5.11)
	if err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	got, err := svc.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("GetGauge() error = %v", err)
	}

	want := 5.11
	if got != want {
		t.Fatalf("GetGauge() got = %v, want %v", got, want)
	}
}

func TestMetricsService_UpdateAndGetCounter(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	got, err := svc.UpdateCounter(ctx, "PollCount", 2)
	if err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	want := int64(2)
	if got != want {
		t.Fatalf("UpdateCounter() got = %v, want %v", got, want)
	}

	got, err = svc.UpdateCounter(ctx, "PollCount", 3)
	if err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	want = 5
	if got != want {
		t.Fatalf("UpdateCounter() got = %v, want %v", got, want)
	}

	stored, err := svc.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("GetCounter() error = %v", err)
	}

	if stored != want {
		t.Fatalf("GetCounter() got = %v, want %v", stored, want)
	}
}

func TestMetricsService_UpdateMetricsAndSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	gaugeValue := 5.11
	counterDelta := int64(42)

	metrics := []model.Metrics{
		{
			ID:    "HeapAlloc",
			MType: model.Gauge,
			Value: &gaugeValue,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &counterDelta,
		},
	}

	if err := svc.UpdateMetrics(ctx, metrics); err != nil {
		t.Fatalf("UpdateMetrics() error = %v", err)
	}

	gauges, counters, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if got, want := gauges["HeapAlloc"], gaugeValue; got != want {
		t.Fatalf("Snapshot() gauge got = %v, want %v", got, want)
	}

	if got, want := counters["PollCount"], counterDelta; got != want {
		t.Fatalf("Snapshot() counter got = %v, want %v", got, want)
	}
}
