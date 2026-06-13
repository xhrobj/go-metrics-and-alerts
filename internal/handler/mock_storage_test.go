package handler_test

import (
	"context"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// mockStorage — минимальная реализация repository.ServerStorage для тестов хэндлера
type mockServerStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newMockServerStorage() *mockServerStorage {
	return &mockServerStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *mockServerStorage) UpdateGauge(_ context.Context, name string, value float64) error {
	m.gauges[name] = value
	return nil
}

func (m *mockServerStorage) UpdateCounter(_ context.Context, name string, delta int64) (int64, error) {
	m.counters[name] += delta
	return m.counters[name], nil
}

func (m *mockServerStorage) UpdateMetrics(_ context.Context, metrics []model.Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			m.gauges[metric.ID] = *metric.Value
		case model.Counter:
			m.counters[metric.ID] += *metric.Delta
		}
	}
	return nil
}

func (m *mockServerStorage) GetGauge(_ context.Context, name string) (float64, error) {
	v, ok := m.gauges[name]
	if !ok {
		return 0, repository.ErrMetricNotFound
	}
	return v, nil
}

func (m *mockServerStorage) GetCounter(_ context.Context, name string) (int64, error) {
	v, ok := m.counters[name]
	if !ok {
		return 0, repository.ErrMetricNotFound
	}
	return v, nil
}

func (m *mockServerStorage) Snapshot(_ context.Context) (map[string]float64, map[string]int64, error) {
	return m.gauges, m.counters, nil
}
