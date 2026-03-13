package service

import (
	"fmt"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// MetricsStorage описывает интерфейс хранилища метрик,
// используемого сервисом.
type MetricsStorage interface {
	UpdateGauge(string, float64)
	UpdateCounter(string, int64)

	GetGauge(string) (float64, error)
	GetCounter(string) (int64, error)

	Snapshot() (map[string]float64, map[string]int64)
}

// Saver описывает механизм сохранения состояния метрик
// во внешнее хранилище (например, файл).
type Saver interface {
	Save(repo repository.Snapshotter) error
}

// MetricsService реализует бизнес-логику работы с метриками.
type MetricsService struct {
	repo            MetricsStorage
	syncPersistence Saver
}

// NewMetricsService создаёт новый сервис метрик,
// использующий переданное хранилище.
func NewMetricsService(repo MetricsStorage) *MetricsService {
	return &MetricsService{
		repo: repo,
	}
}

// EnableSyncSave включает синхронное сохранение метрик
// после каждого изменения, используя переданный Saver.
func (m *MetricsService) EnableSyncSave(syncPersistence Saver) {
	m.syncPersistence = syncPersistence
}

// UpdateGauge сохраняет значение gauge-метрики.
func (m *MetricsService) UpdateGauge(metricName string, value float64) error {
	m.repo.UpdateGauge(metricName, value)

	if err := m.saveIfSync(); err != nil {
		return fmt.Errorf("sync save failed: %w", err)
	}
	return nil
}

// UpdateCounter увеличивает значение counter-метрики на delta
// и возвращает итоговое значение счётчика.
func (m *MetricsService) UpdateCounter(metricName string, delta int64) (int64, error) {
	m.repo.UpdateCounter(metricName, delta)

	if err := m.saveIfSync(); err != nil {
		return 0, fmt.Errorf("sync save failed: %w", err)
	}

	total, err := m.repo.GetCounter(metricName)
	if err != nil {
		return 0, fmt.Errorf("get counter: %w", err)
	}

	return total, nil
}

// GetGauge возвращает текущее значение gauge-метрики.
func (m *MetricsService) GetGauge(metricName string) (float64, error) {
	return m.repo.GetGauge(metricName)
}

// GetCounter возвращает текущее значение counter-метрики.
func (m *MetricsService) GetCounter(metricName string) (int64, error) {
	return m.repo.GetCounter(metricName)
}

// Snapshot возвращает копию всех метрик,
// сохранённых в хранилище.
func (m *MetricsService) Snapshot() (map[string]float64, map[string]int64) {
	return m.repo.Snapshot()
}

func (m *MetricsService) saveIfSync() error {
	if m.syncPersistence == nil {
		return nil
	}

	return m.syncPersistence.Save(m.repo)
}
