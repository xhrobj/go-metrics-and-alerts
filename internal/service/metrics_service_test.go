package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
)

type saverFunc func(context.Context, repository.Snapshotter) error

func (f saverFunc) Save(ctx context.Context, repo repository.Snapshotter) error {
	return f(ctx, repo)
}

type metricsStorageStub struct {
	updateGaugeErr   error
	updateCounter    int64
	updateCounterErr error
	updateMetricsErr error
	getGauge         float64
	getGaugeErr      error
	getCounter       int64
	getCounterErr    error
	snapshotGauges   map[string]float64
	snapshotCounters map[string]int64
	snapshotErr      error
}

func (s *metricsStorageStub) UpdateGauge(
	context.Context,
	string,
	float64,
) error {
	return s.updateGaugeErr
}

func (s *metricsStorageStub) UpdateCounter(
	context.Context,
	string,
	int64,
) (int64, error) {
	return s.updateCounter, s.updateCounterErr
}

func (s *metricsStorageStub) UpdateMetrics(
	context.Context,
	[]model.Metrics,
) error {
	return s.updateMetricsErr
}

func (s *metricsStorageStub) GetGauge(
	context.Context,
	string,
) (float64, error) {
	return s.getGauge, s.getGaugeErr
}

func (s *metricsStorageStub) GetCounter(
	context.Context,
	string,
) (int64, error) {
	return s.getCounter, s.getCounterErr
}

func (s *metricsStorageStub) Snapshot(
	context.Context,
) (map[string]float64, map[string]int64, error) {
	return s.snapshotGauges, s.snapshotCounters, s.snapshotErr
}

func TestMetricsService_UpdateAndGetGauge(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	err := svc.UpdateGauge(ctx, "Alloc", 5.11)
	if err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	got, err := svc.GetGauge(ctx, "Alloc")
	if err != nil {
		t.Fatalf("GetGauge() error = %v", err)
	}

	want := 5.11
	if got != want {
		t.Fatalf("GetGauge() got = %v, want %v", got, want)
	}
}

func TestMetricsService_UpdateAndGetCounter(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	got, err := svc.UpdateCounter(ctx, "PollCount", 2)
	if err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	want := int64(2)
	if got != want {
		t.Fatalf("UpdateCounter() got = %v, want %v", got, want)
	}

	got, err = svc.UpdateCounter(ctx, "PollCount", 3)
	if err != nil {
		t.Fatalf("UpdateCounter() error = %v", err)
	}

	want = 5
	if got != want {
		t.Fatalf("UpdateCounter() got = %v, want %v", got, want)
	}

	stored, err := svc.GetCounter(ctx, "PollCount")
	if err != nil {
		t.Fatalf("GetCounter() error = %v", err)
	}

	if stored != want {
		t.Fatalf("GetCounter() got = %v, want %v", stored, want)
	}
}

func TestMetricsService_UpdateMetricsAndSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	gaugeValue := 5.11
	counterDelta := int64(42)

	metrics := []model.Metrics{
		{
			ID:    "HeapAlloc",
			MType: model.Gauge,
			Value: &gaugeValue,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &counterDelta,
		},
	}

	if err := svc.UpdateMetrics(ctx, metrics); err != nil {
		t.Fatalf("UpdateMetrics() error = %v", err)
	}

	gauges, counters, err := svc.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}

	if got, want := gauges["HeapAlloc"], gaugeValue; got != want {
		t.Fatalf("Snapshot() gauge got = %v, want %v", got, want)
	}

	if got, want := counters["PollCount"], counterDelta; got != want {
		t.Fatalf("Snapshot() counter got = %v, want %v", got, want)
	}
}

func TestMetricsService_SyncSavePassesContext(t *testing.T) {
	type contextKey struct{}

	key := contextKey{}
	ctx := context.WithValue(context.Background(), key, "some-value")

	repo := repository.NewMemStorage()
	svc := service.NewMetricsService(repo)

	var got string

	svc.EnableSyncSave(
		saverFunc(func(ctx context.Context, _ repository.Snapshotter) error {
			got, _ = ctx.Value(key).(string)
			return nil
		}),
	)

	if err := svc.UpdateGauge(ctx, "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	want := "some-value"
	if got != want {
		t.Fatalf("Save() context value = %q, want %q", got, want)
	}
}

func TestMetricsService_UpdateRepositoryError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("repository error")

	tests := []struct {
		name   string
		repo   *metricsStorageStub
		update func(*service.MetricsService) error
	}{
		{
			name: "gauge",
			repo: &metricsStorageStub{
				updateGaugeErr: wantErr,
			},
			update: func(svc *service.MetricsService) error {
				return svc.UpdateGauge(ctx, "Alloc", 5.11)
			},
		},
		{
			name: "counter",
			repo: &metricsStorageStub{
				updateCounterErr: wantErr,
			},
			update: func(svc *service.MetricsService) error {
				_, err := svc.UpdateCounter(ctx, "PollCount", 42)
				return err
			},
		},
		{
			name: "metrics batch",
			repo: &metricsStorageStub{
				updateMetricsErr: wantErr,
			},
			update: func(svc *service.MetricsService) error {
				value := 5.11

				return svc.UpdateMetrics(ctx, []model.Metrics{
					{
						ID:    "Alloc",
						MType: model.Gauge,
						Value: &value,
					},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMetricsService(tt.repo)

			err := tt.update(svc)
			if !errors.Is(err, wantErr) {
				t.Fatalf("update error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestMetricsService_SyncSaveError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("save error")

	tests := []struct {
		name   string
		update func(*service.MetricsService) error
	}{
		{
			name: "gauge",
			update: func(svc *service.MetricsService) error {
				return svc.UpdateGauge(ctx, "Alloc", 5.11)
			},
		},
		{
			name: "counter",
			update: func(svc *service.MetricsService) error {
				_, err := svc.UpdateCounter(ctx, "PollCount", 42)
				return err
			},
		},
		{
			name: "metrics batch",
			update: func(svc *service.MetricsService) error {
				value := 5.11

				return svc.UpdateMetrics(ctx, []model.Metrics{
					{
						ID:    "Alloc",
						MType: model.Gauge,
						Value: &value,
					},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemStorage()
			svc := service.NewMetricsService(repo)

			svc.EnableSyncSave(
				saverFunc(func(
					context.Context,
					repository.Snapshotter,
				) error {
					return wantErr
				}),
			)

			err := tt.update(svc)
			if !errors.Is(err, wantErr) {
				t.Fatalf("update error = %v, want %v", err, wantErr)
			}
		})
	}
}

func TestMetricsService_GetMetricError(t *testing.T) {
	ctx := context.Background()
	storageErr := errors.New("storage error")

	tests := []struct {
		name       string
		repo       *metricsStorageStub
		get        func(*service.MetricsService) error
		wantErr    error
		wantDirect bool
	}{
		{
			name: "gauge not found",
			repo: &metricsStorageStub{
				getGaugeErr: repository.ErrMetricNotFound,
			},
			get: func(svc *service.MetricsService) error {
				_, err := svc.GetGauge(ctx, "missing")
				return err
			},
			wantErr:    repository.ErrMetricNotFound,
			wantDirect: true,
		},
		{
			name: "gauge storage error",
			repo: &metricsStorageStub{
				getGaugeErr: storageErr,
			},
			get: func(svc *service.MetricsService) error {
				_, err := svc.GetGauge(ctx, "Alloc")
				return err
			},
			wantErr: storageErr,
		},
		{
			name: "counter not found",
			repo: &metricsStorageStub{
				getCounterErr: repository.ErrMetricNotFound,
			},
			get: func(svc *service.MetricsService) error {
				_, err := svc.GetCounter(ctx, "missing")
				return err
			},
			wantErr:    repository.ErrMetricNotFound,
			wantDirect: true,
		},
		{
			name: "counter storage error",
			repo: &metricsStorageStub{
				getCounterErr: storageErr,
			},
			get: func(svc *service.MetricsService) error {
				_, err := svc.GetCounter(ctx, "PollCount")
				return err
			},
			wantErr: storageErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMetricsService(tt.repo)

			err := tt.get(svc)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("get metric error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantDirect && err != tt.wantErr {
				t.Fatalf(
					"get metric error = %v, want direct error %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestMetricsService_SnapshotError(t *testing.T) {
	wantErr := errors.New("snapshot error")
	repo := &metricsStorageStub{
		snapshotErr: wantErr,
	}
	svc := service.NewMetricsService(repo)

	_, _, err := svc.Snapshot(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Snapshot() error = %v, want %v", err, wantErr)
	}
}
