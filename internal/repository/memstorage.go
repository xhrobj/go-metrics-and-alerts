package repository

import (
	"errors"
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
func (m *MemStorage) UpdateGauge(metricName string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[metricName] = value
}

// UpdateCounter увеличивает значение counter-метрики на delta.
func (m *MemStorage) UpdateCounter(metricName string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[metricName] += delta
}

// GetGauge возвращает значение gauge-метрики.
func (m *MemStorage) GetGauge(metricName string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, saved := m.gauges[metricName]
	if !saved {
		return 0, errors.New("no data")
	}

	return value, nil
}

// GetCounter возвращает значение counter-метрики.
func (m *MemStorage) GetCounter(metricName string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total, saved := m.counters[metricName]
	if !saved {
		return 0, errors.New("no data")
	}

	return total, nil
}

// Snapshot возвращает копию всех метрик.
func (m *MemStorage) Snapshot() (gaugesCopy map[string]float64, countersCopy map[string]int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gaugesCopy = make(map[string]float64, len(m.gauges))
	for k, v := range m.gauges {
		gaugesCopy[k] = v
	}

	countersCopy = make(map[string]int64, len(m.counters))
	for k, v := range m.counters {
		countersCopy[k] = v
	}

	return
}

// SetGauge устанавливает значение gauge-метрики.
func (m *MemStorage) SetGauge(metricName string, value float64) {
	m.UpdateGauge(metricName, value)
}

// SetCounter устанавливает значение counter-метрики.
func (m *MemStorage) SetCounter(metricName string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[metricName] = delta
}
