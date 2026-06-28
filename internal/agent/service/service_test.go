package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
)

type snapshotErrorStorage struct {
	err error
}

func (s *snapshotErrorStorage) UpdateGauge(context.Context, string, float64) error {
	return nil
}

func (s *snapshotErrorStorage) Snapshot(
	context.Context,
) (map[string]float64, map[string]int64, error) {
	return nil, nil, s.err
}

// TestReportingServicePreparePeriodicReport проверяет формирование отчёта
// и сброс накопленного PollCount.
func TestReportingServicePreparePeriodicReport(t *testing.T) {
	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	service := New(repo, newNoopSender(), zap.NewNop())
	service.pollSinceReport.Store(3)

	report, ok, err := service.PreparePeriodicReport(context.Background())
	if err != nil {
		t.Fatalf("PreparePeriodicReport() error = %v, want nil", err)
	}
	if !ok {
		t.Fatal("PreparePeriodicReport() ok = false, want true")
	}

	assertMetricValue(t, report.Metrics, "Alloc", 5.11)
	assertMetricDelta(t, report.Metrics, "PollCount", 3)

	if got := service.pollSinceReport.Load(); got != 0 {
		t.Fatalf("pollSinceReport = %d, want 0", got)
	}
}

// TestReportingServiceFlushRestoresPollCountOnSnapshotError проверяет,
// что сервис возвращает PollCount при ошибке финального snapshot.
func TestReportingServiceFlushRestoresPollCountOnSnapshotError(t *testing.T) {
	wantErr := errors.New("snapshot failed")
	service := New(
		&snapshotErrorStorage{err: wantErr},
		newNoopSender(),
		zap.NewNop(),
	)
	service.pollSinceReport.Store(42)

	err := service.Flush(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("Flush() error = %v, want error wrapping %v", err, wantErr)
	}
	if got, want := service.pollSinceReport.Load(), int64(42); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}

// TestReportingServiceFlushRestoresPollCountOnSendError проверяет,
// что сервис возвращает PollCount при ошибке финальной отправки.
func TestReportingServiceFlushRestoresPollCountOnSendError(t *testing.T) {
	wantErr := errors.New("send failed")
	sender := senderFunc(func(context.Context, []model.Metrics) error {
		return wantErr
	})

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	service := New(repo, sender, zap.NewNop())
	service.pollSinceReport.Store(42)

	err := service.Flush(context.Background())
	if err == nil {
		t.Fatal("Flush() error = nil, want error")
	}
	if got, want := err.Error(), "send final metrics batch"; !strings.Contains(got, want) {
		t.Fatalf("Flush() error = %q, want error containing %q", got, want)
	}
	if got, want := service.pollSinceReport.Load(), int64(42); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}

// TestReportingServiceSendRestoresOnlyFailedReportPollCount проверяет,
// что ошибка отправки не затирает новые poll'ы, накопленные параллельно.
func TestReportingServiceSendRestoresOnlyFailedReportPollCount(t *testing.T) {
	wantErr := errors.New("send failed")
	service := New(
		repository.NewMemStorage(),
		senderFunc(func(context.Context, []model.Metrics) error {
			return wantErr
		}),
		zap.NewNop(),
	)

	report := Report{PollCount: 42}
	service.pollSinceReport.Store(7)

	err := service.Send(context.Background(), report)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Send() error = %v, want %v", err, wantErr)
	}
	if got, want := service.pollSinceReport.Load(), int64(49); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}

// TestReportingServicePrepareFinalReportIncludesZeroPollCount фиксирует,
// что shutdown-отчёт отправляет snapshot даже без новых runtime-poll'ов.
func TestReportingServicePrepareFinalReportIncludesZeroPollCount(t *testing.T) {
	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "TotalMemory", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	service := New(repo, newNoopSender(), zap.NewNop())

	report, err := service.prepareFinalReport(context.Background())
	if err != nil {
		t.Fatalf("prepareFinalReport() error = %v, want nil", err)
	}

	assertMetricValue(t, report.Metrics, "TotalMemory", 5.11)
	assertMetricDelta(t, report.Metrics, "PollCount", 0)
}
