package repository

import (
	"errors"
	"sync"
)

// MemStorage хранит метрики в памяти.
type MemStorage struct {
	mu       sync.Mutex
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
func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.gauges[name] = value
}

// UpdateCounter увеличивает значение counter-метрики на delta.
func (m *MemStorage) UpdateCounter(name string, delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] += delta
}

// ResetCounter сбрасывает значение counter-метрики в 0.
func (m *MemStorage) ResetCounter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] = 0
}

// GetGauge возвращает значение gauge-метрики.
func (m *MemStorage) GetGauge(name string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, saved := m.gauges[name]
	if !saved {
		return 0, errors.New("no data")
	}

	return value, nil
}

// GetCounter возвращает значение counter-метрики.
func (m *MemStorage) GetCounter(name string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, saved := m.counters[name]
	if !saved {
		return 0, errors.New("no data")
	}

	return value, nil
}

// Snapshot возвращает копию всех метрик.
func (m *MemStorage) Snapshot() (gaugesCopy map[string]float64, countersCopy map[string]int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

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

// SetGauge восстанавливает значение gauge-метрики.
func (m *MemStorage) SetGauge(name string, value float64) {
	m.UpdateGauge(name, value)
}

// SetCounter восстанавливает значение counter-метрики.
func (m *MemStorage) SetCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters[name] = value
}
