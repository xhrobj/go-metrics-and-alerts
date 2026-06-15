package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
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

// TestNewReturnsErrorForInvalidCryptoKey проверяет, что Агент
// не создаётся, если публичный ключ не удалось загрузить.
func TestNewReturnsErrorForInvalidCryptoKey(t *testing.T) {
	cfg := config.AgentConfig{
		ServerAddr:          "localhost:8080",
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
		CryptoKey:           filepath.Join(t.TempDir(), "missing-public.pem"),
	}

	_, err := New(repository.NewMemStorage(), cfg, zap.NewNop())
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}

	if got, want := err.Error(), "load public key"; !strings.Contains(got, want) {
		t.Fatalf("New() error = %q, want error containing %q", got, want)
	}
}

// TestFlushRestoresPollCountOnSnapshotError проверяет, что при ошибке получения
// финального снимка Агент возвращает PollCount для следующей попытки отправки.
func TestFlushRestoresPollCountOnSnapshotError(t *testing.T) {
	wantErr := errors.New("snapshot failed")
	repo := &snapshotErrorStorage{err: wantErr}

	cfg := config.AgentConfig{
		ServerAddr:          "localhost:8080",
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}

	a, err := New(repo, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	a.pollSinceReport.Store(42)

	err = a.flush(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("flush() error = %v, want error wrapping %v", err, wantErr)
	}

	if got, want := a.pollSinceReport.Load(), int64(42); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}

// TestFlushRestoresPollCountOnSendError проверяет, что при ошибке финальной
// отправки Агент возвращает PollCount для следующей попытки.
func TestFlushRestoresPollCountOnSendError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		rs http.ResponseWriter,
		_ *http.Request,
	) {
		rs.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v, want nil", err)
	}

	cfg := config.AgentConfig{
		ServerAddr:          server.URL,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}

	a, err := New(repo, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	a.pollSinceReport.Store(42)

	err = a.flush(context.Background())
	if err == nil {
		t.Fatal("flush() error = nil, want error")
	}

	if got, want := err.Error(), "send final metrics batch"; !strings.Contains(got, want) {
		t.Fatalf("flush() error = %q, want error containing %q", got, want)
	}

	if got, want := a.pollSinceReport.Load(), int64(42); got != want {
		t.Fatalf("pollSinceReport = %d, want %d", got, want)
	}
}

// TestRunDrainsQueueAndSendsFinalSnapshot проверяет, что при shutdown Агент
// дожидается активной отправки, опустошает очередь и отправляет финальный снимок.
func TestRunDrainsQueueAndSendsFinalSnapshot(t *testing.T) {
	firstRequestStarted := make(chan struct{})
	allowFirstRequestToFinish := make(chan struct{})
	requestMetrics := make(chan []model.Metrics, 2)

	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(rs http.ResponseWriter, rq *http.Request) {
		zr, err := gzip.NewReader(rq.Body)
		if err != nil {
			http.Error(rs, err.Error(), http.StatusBadRequest)
			return
		}
		defer func() {
			_ = zr.Close()
		}()

		var metrics []model.Metrics
		if err := json.NewDecoder(zr).Decode(&metrics); err != nil {
			http.Error(rs, err.Error(), http.StatusBadRequest)
			return
		}

		requestMetrics <- metrics

		if requestCount.Add(1) == 1 {
			close(firstRequestStarted)
			<-allowFirstRequestToFinish
		}

		rs.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		select {
		case <-allowFirstRequestToFinish:
		default:
			close(allowFirstRequestToFinish)
		}

		server.Close()
	})

	cfg := config.AgentConfig{
		ServerAddr:          server.URL,
		PollIntervalInSec:   3600,
		ReportIntervalInSec: 3600,
		RateLimit:           1,
	}

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}

	a, err := New(repo, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	a.pollSinceReport.Store(42)
	a.report()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- a.Run(ctx)
	}()

	select {
	case <-firstRequestStarted:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for active request")
	}

	if err := repo.UpdateGauge(context.Background(), "Alloc", 6.11); err != nil {
		t.Fatalf("UpdateGauge() error = %v", err)
	}
	a.pollSinceReport.Store(7)

	cancel()

	select {
	case err := <-runErrCh:
		t.Fatalf("Run() returned before active request completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(allowFirstRequestToFinish)

	var gotBatches [][]model.Metrics
	for range 2 {
		select {
		case metrics := <-requestMetrics:
			gotBatches = append(gotBatches, metrics)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for metrics request")
		}
	}

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run() to stop")
	}

	if got, want := requestCount.Load(), int32(2); got != want {
		t.Fatalf("request count = %d, want %d", got, want)
	}

	assertMetricValue(t, gotBatches[0], "Alloc", 5.11)
	assertMetricDelta(t, gotBatches[0], "PollCount", 42)
	assertMetricValue(t, gotBatches[1], "Alloc", 6.11)
	assertMetricDelta(t, gotBatches[1], "PollCount", 7)
}

func assertMetricValue(
	t *testing.T,
	metrics []model.Metrics,
	name string,
	want float64,
) {
	t.Helper()

	for _, metric := range metrics {
		if metric.ID == name && metric.MType == model.Gauge {
			if metric.Value == nil {
				t.Fatalf("metric %q value = nil, want %v", name, want)
			}

			if got := *metric.Value; got != want {
				t.Fatalf("metric %q value = %v, want %v", name, got, want)
			}

			return
		}
	}

	t.Fatalf("metric %q not found", name)
}

func assertMetricDelta(t *testing.T, metrics []model.Metrics, name string, want int64) {
	t.Helper()

	for _, metric := range metrics {
		if metric.ID == name && metric.MType == model.Counter {
			if metric.Delta == nil {
				t.Fatalf("metric %q delta = nil, want %d", name, want)
			}

			if got := *metric.Delta; got != want {
				t.Fatalf("metric %q delta = %d, want %d", name, got, want)
			}

			return
		}
	}

	t.Fatalf("metric %q not found", name)
}
