package handler_test

import "errors"

var errNoData = errors.New("no data")

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

func (m *mockServerStorage) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *mockServerStorage) UpdateCounter(name string, delta int64) {
	m.counters[name] += delta
}

func (m *mockServerStorage) Snapshot() (map[string]float64, map[string]int64) {
	return m.gauges, m.counters
}

func (m *mockServerStorage) GetGauge(name string) (float64, error) {
	v, ok := m.gauges[name]
	if !ok {
		return 0, errNoData
	}
	return v, nil
}

func (m *mockServerStorage) GetCounter(name string) (int64, error) {
	v, ok := m.counters[name]
	if !ok {
		return 0, errNoData
	}
	return v, nil
}
