package repository

import (
	"encoding/json"
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
