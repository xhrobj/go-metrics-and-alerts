package repository

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

// Snapshotter описывает хранилище, из которого можно получить снимок метрик.
type Snapshotter interface {
	Snapshot() (map[string]float64, map[string]int64)
}

// FileStorage отвечает за сохранение метрик в файл.
type FileStorage struct {
	path string
}

// Restorer описывает хранилище, в которое можно восстановить метрики.
type Restorer interface {
	SetGauge(name string, value float64)
	SetCounter(name string, value int64)
}

// NewFileStorage создаёт файловое хранилище метрик.
func NewFileStorage(path string) *FileStorage {
	return &FileStorage{path: path}
}

// Save сохраняет все текущие метрики в файл в формате JSON.
func (f *FileStorage) Save(repo Snapshotter) error {
	gauges, counters := repo.Snapshot()

	metrics := make([]model.Metrics, 0, len(gauges)+len(counters))

	for name, value := range gauges {
		v := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}

	for name, value := range counters {
		v := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &v,
		})
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	return os.WriteFile(f.path, data, 0666)
}

// Load загружает метрики из файла в хранилище.
func (f *FileStorage) Load(repo Restorer) error {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	if len(data) == 0 {
		return nil
	}

	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if metric.Value != nil {
				repo.SetGauge(metric.ID, *metric.Value)
			}
		case model.Counter:
			if metric.Delta != nil {
				repo.SetCounter(metric.ID, *metric.Delta)
			}
		}
	}

	return nil
}
