package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

func TestMemStorage_UpdateAndGetMetrics(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()

	if err := storage.UpdateGauge(ctx, "Alloc", 123.45); err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	gotGauge, err := storage.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("GetGauge() error = %v", err)
	}

	wantGauge := 123.45
	if gotGauge != wantGauge {
		t.Fatalf("GetGauge() got = %v, want %v", gotGauge, wantGauge)
	}

	if err := storage.UpdateCounter(ctx, "PollCount", 2); err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	if err := storage.UpdateCounter(ctx, "PollCount", 3); err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	gotCounter, err := storage.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("GetCounter() error = %v", err)
	}

	wantCounter := int64(5)
	if gotCounter != wantCounter {
		t.Fatalf("GetCounter() got = %v, want %v", gotCounter, wantCounter)
	}
}

func TestMemStorage_GetMissingMetric(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()

	_, err := storage.GetGauge(ctx, "missing-gauge")
	if !errors.Is(err, repository.ErrMetricNotFound) {
		t.Fatalf("GetGauge() error = %v, want %v", err, repository.ErrMetricNotFound)
	}

	_, err = storage.GetCounter(ctx, "missing-counter")
	if !errors.Is(err, repository.ErrMetricNotFound) {
		t.Fatalf("GetCounter() error = %v, want %v", err, repository.ErrMetricNotFound)
	}
}

func TestMemStorage_UpdateMetricsAndSnapshot(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()

	gaugeValue := 1024.5
	counterDelta := int64(7)

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

	if err := storage.UpdateMetrics(ctx, metrics); err != nil {
		t.Fatalf("UpdateMetrics() error = %v", err)
	}

	gauges, counters, err := storage.Snapshot(ctx)
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

func TestMemStorage_SetMetrics(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemStorage()

	storage.SetGauge("RestoredGauge", 42.5)
	storage.SetCounter("RestoredCounter", 11)

	gotGauge, err := storage.GetGauge(ctx, "RestoredGauge")
	if err != nil {
		t.Fatalf("GetGauge() error = %v", err)
	}

	wantGauge := 42.5
	if gotGauge != wantGauge {
		t.Fatalf("GetGauge() got = %v, want %v", gotGauge, wantGauge)
	}

	gotCounter, err := storage.GetCounter(ctx, "RestoredCounter")
	if err != nil {
		t.Fatalf("GetCounter() error = %v", err)
	}

	wantCounter := int64(11)
	if gotCounter != wantCounter {
		t.Fatalf("GetCounter() got = %v, want %v", gotCounter, wantCounter)
	}
}
