package repository

import (
	"context"
	"sync"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

// MemStorage хранит метрики в памяти.
type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewMemStorage создаёт новое in-memory хранилище метрик.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// UpdateGauge сохраняет значение gauge-метрики.
func (m *MemStorage) UpdateGauge(_ context.Context, metricName string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[metricName] = value

	return nil
}

// UpdateCounter увеличивает значение counter-метрики на delta
// и возвращает итоговое значение счётчика.
func (m *MemStorage) UpdateCounter(_ context.Context, metricName string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[metricName] += delta

	return m.counters[metricName], nil
}

// UpdateMetrics пакетно обновляет метрики под одной блокировкой.
func (m *MemStorage) UpdateMetrics(_ context.Context, metrics []model.Metrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

// GetGauge возвращает значение gauge-метрики.
// Если метрика не найдена, возвращается ErrMetricNotFound.
func (m *MemStorage) GetGauge(_ context.Context, metricName string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, saved := m.gauges[metricName]
	if !saved {
		return 0, ErrMetricNotFound
	}

	return value, nil
}

// GetCounter возвращает значение counter-метрики.
// Если метрика не найдена, возвращается ErrMetricNotFound.
func (m *MemStorage) GetCounter(_ context.Context, metricName string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total, saved := m.counters[metricName]
	if !saved {
		return 0, ErrMetricNotFound
	}

	return total, nil
}

// Snapshot возвращает снимок (копию) всех метрик.
// Результат разделяется на две map: gauges и counters.
func (m *MemStorage) Snapshot(_ context.Context) (map[string]float64, map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gaugesCopy := make(map[string]float64, len(m.gauges))
	for k, v := range m.gauges {
		gaugesCopy[k] = v
	}

	countersCopy := make(map[string]int64, len(m.counters))
	for k, v := range m.counters {
		countersCopy[k] = v
	}

	return gaugesCopy, countersCopy, nil
}

// SetGauge устанавливает значение gauge-метрики.
func (m *MemStorage) SetGauge(metricName string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[metricName] = value
}

// SetCounter устанавливает значение counter-метрики.
func (m *MemStorage) SetCounter(metricName string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[metricName] = delta
}
