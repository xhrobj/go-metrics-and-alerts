package agent

import (
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

// report() делает POST и выставляет требуемый Content-Type
func TestAgent_Report_SendsPOSTWithContentType(t *testing.T) {
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/update" {
			t.Fatalf("expected path /update, got %q", r.URL.Path)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %q", ct)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repository.NewMemStorage()
	repo.UpdateGauge("Alloc", 5.42)
	repo.UpdateCounter("PollCount", 1)

	a, _ := New(repo, server.URL, 2, 10)
	a.report()

	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
}

// report() отправляет корректные JSON-метрики на /update
func TestAgent_Report_SendsCorrectJSONMetrics(t *testing.T) {
	var metrics []model.Metrics

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update" {
			t.Fatalf("expected path /update, got %q", r.URL.Path)
		}

		if got := r.Header.Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("expected Content-Encoding gzip, got %q", got)
		}

		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer zr.Close()

		var m model.Metrics
		if err := json.NewDecoder(zr).Decode(&m); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		metrics = append(metrics, m)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repository.NewMemStorage()
	repo.UpdateGauge("Alloc", 5.42)

	a, _ := New(repo, server.URL, 2, 10)
	a.pollSinceReport = 3
	a.report()

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
			if *m.Value != 5.42 {
				t.Fatalf("expected gauge value 5.42, got %v", *m.Value)
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

	if a.pollSinceReport != 0 {
		t.Errorf("expected pollSinceReport to be reset to 0, got %d", a.pollSinceReport)
	}
}
