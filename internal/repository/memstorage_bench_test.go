package repository_test

import (
	"context"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

func BenchmarkMemStorage_UpdateMetrics(b *testing.B) {
	ctx := context.Background()
	storage := repository.NewMemStorage()
	metrics := benchMetrics()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := storage.UpdateMetrics(ctx, metrics); err != nil {
			b.Fatalf("UpdateMetrics() error = %v", err)
		}
	}
}

func BenchmarkMemStorage_Snapshot(b *testing.B) {
	ctx := context.Background()
	storage := repository.NewMemStorage()
	metrics := benchMetrics()

	if err := storage.UpdateMetrics(ctx, metrics); err != nil {
		b.Fatalf("UpdateMetrics() error = %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		gauges, counters, err := storage.Snapshot(ctx)
		if err != nil {
			b.Fatalf("Snapshot() error = %v", err)
		}
		if len(gauges) == 0 || len(counters) == 0 {
			b.Fatalf("Snapshot() returned empty data")
		}
	}
}

func benchMetrics() []model.Metrics {
	return []model.Metrics{
		gaugeMetric("Alloc", 120000),
		gaugeMetric("BuckHashSys", 1450),
		gaugeMetric("Frees", 3000),
		gaugeMetric("GCCPUFraction", 0.000012),
		gaugeMetric("GCSys", 350000),
		gaugeMetric("HeapAlloc", 900000),
		gaugeMetric("HeapIdle", 700000),
		gaugeMetric("HeapInuse", 800000),
		gaugeMetric("HeapObjects", 4200),
		gaugeMetric("HeapReleased", 200000),
		gaugeMetric("HeapSys", 1600000),
		gaugeMetric("LastGC", 123456789),
		gaugeMetric("Lookups", 0),
		gaugeMetric("MCacheInuse", 2400),
		gaugeMetric("MCacheSys", 15600),
		gaugeMetric("MSpanInuse", 52000),
		gaugeMetric("MSpanSys", 65000),
		gaugeMetric("Mallocs", 7200),
		gaugeMetric("NextGC", 4194304),
		gaugeMetric("NumForcedGC", 0),
		gaugeMetric("NumGC", 12),
		gaugeMetric("OtherSys", 500000),
		gaugeMetric("PauseTotalNs", 1000000),
		gaugeMetric("StackInuse", 327680),
		gaugeMetric("StackSys", 327680),
		gaugeMetric("Sys", 6000000),
		gaugeMetric("TotalAlloc", 1500000),
		gaugeMetric("TotalMemory", 16000000000),
		gaugeMetric("FreeMemory", 7000000000),
		gaugeMetric("CPUutilization1", 12.5),
		counterMetric("PollCount", 1),
		counterMetric("RandomValue", 42),
	}
}

func gaugeMetric(id string, value float64) model.Metrics {
	return model.Metrics{
		ID:    id,
		MType: model.Gauge,
		Value: &value,
	}
}

func counterMetric(id string, delta int64) model.Metrics {
	return model.Metrics{
		ID:    id,
		MType: model.Counter,
		Delta: &delta,
	}
}
