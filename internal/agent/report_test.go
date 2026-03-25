package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// report() ставит задачу в очередь, а воркер отправляет POST с нужными заголовками.
func TestAgent_Report_SendsPOSTWithContentType(t *testing.T) {
	requests := 0
	done := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/updates" {
			t.Fatalf("expected path /updates, got %q", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}

		ce := r.Header.Get("Content-Encoding")
		if ce != "gzip" {
			t.Fatalf("expected Content-Encoding gzip, got %q", ce)
		}

		w.WriteHeader(http.StatusOK)
		done <- struct{}{}
	}))
	defer server.Close()

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("failed to prepare test gauge metric: %v", err)
	}

	a, err := New(repo, server.URL, 2, 10, 5, "secret-key")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	go a.runSendWorker()

	a.pollSinceReport.Store(1)
	a.report()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for request")
	}

	if requests != 1 {
		t.Fatalf("expected 1 requests, got %d", requests)
	}
}

// report() ставит в очередь batch, а воркер отправляет корректные JSON-метрики на /updates.
func TestAgent_Report_SendsCorrectJSONMetrics(t *testing.T) {
	var metrics []model.Metrics
	done := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { done <- struct{}{} }()

		if r.URL.Path != "/updates" {
			t.Fatalf("expected path /updates, got %q", r.URL.Path)
		}

		if got := r.Header.Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("expected Content-Encoding gzip, got %q", got)
		}

		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer zr.Close()

		if err := json.NewDecoder(zr).Decode(&metrics); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repository.NewMemStorage()
	if err := repo.UpdateGauge(context.Background(), "Alloc", 5.11); err != nil {
		t.Fatalf("failed to prepare test gauge metric: %v", err)
	}

	a, err := New(repo, server.URL, 2, 10, 5, "secret-key")
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	go a.runSendWorker()

	a.pollSinceReport.Store(3)
	a.report()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for request")
	}

	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}

	var foundGauge bool
	var foundCounter bool

	for _, m := range metrics {
		switch {
		case m.ID == "Alloc" && m.MType == model.Gauge:
			foundGauge = true
			if m.Value == nil {
				t.Fatal("expected gauge value to be set")
			}
			if *m.Value != 5.11 {
				t.Fatalf("expected gauge value 5.11, got %v", *m.Value)
			}
			if m.Delta != nil {
				t.Fatal("expected gauge delta to be nil")
			}

		case m.ID == "PollCount" && m.MType == model.Counter:
			foundCounter = true
			if m.Delta == nil {
				t.Fatal("expected counter delta to be set")
			}
			if *m.Delta != 3 {
				t.Fatalf("expected counter delta 3, got %v", *m.Delta)
			}
			if m.Value != nil {
				t.Fatal("expected counter value to be nil")
			}
		}
	}

	if !foundGauge {
		t.Error("expected gauge metric Alloc to be sent")
	}

	if !foundCounter {
		t.Error("expected counter metric PollCount to be sent")
	}

	if a.pollSinceReport.Load() != 0 {
		t.Errorf("expected pollSinceReport to be reset to 0, got %d", a.pollSinceReport.Load())
	}
}
