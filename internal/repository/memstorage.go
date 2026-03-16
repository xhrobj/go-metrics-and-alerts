package repository

import (
	"sync"
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
func (m *MemStorage) UpdateGauge(metricName string, value float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[metricName] = value

	return nil
}

// UpdateCounter увеличивает значение counter-метрики на delta.
func (m *MemStorage) UpdateCounter(metricName string, delta int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[metricName] += delta

	return nil
}

// GetGauge возвращает значение gauge-метрики.
func (m *MemStorage) GetGauge(metricName string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, saved := m.gauges[metricName]
	if !saved {
		return 0, ErrMetricNotFound
	}

	return value, nil
}

// GetCounter возвращает значение counter-метрики.
func (m *MemStorage) GetCounter(metricName string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total, saved := m.counters[metricName]
	if !saved {
		return 0, ErrMetricNotFound
	}

	return total, nil
}

// Snapshot возвращает копию всех метрик.
func (m *MemStorage) Snapshot() (map[string]float64, map[string]int64, error) {
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
	_ = m.UpdateGauge(metricName, value)
}

// SetCounter устанавливает значение counter-метрики.
func (m *MemStorage) SetCounter(metricName string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[metricName] = delta
}
