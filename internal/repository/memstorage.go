package repository

import "errors"

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *MemStorage) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
}

func (m *MemStorage) UpdateCounter(name string, delta int64) {
	m.counters[name] += delta
}

func (m *MemStorage) GetGauge(name string) (float64, error) {
	value, saved := m.gauges[name]
	if !saved {
		return 0, errors.New("no data")
	}
	return value, nil
}

func (m *MemStorage) GetCounter(name string) (int64, error) {
	value, saved := m.counters[name]
	if !saved {
		return 0, errors.New("no data")
	}
	return value, nil
}
