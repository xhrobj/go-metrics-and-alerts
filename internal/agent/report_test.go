package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

		ct := r.Header.Get("Content-Type")
		if ct != "text/plain" {
			t.Fatalf("expected Content-Type text/plain, got %q", ct)
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

// report() формирует корректные URL
func TestAgent_Report_UsesCorrectPaths(t *testing.T) {
	paths := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths[r.URL.Path] = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo := repository.NewMemStorage()
	repo.UpdateGauge("Alloc", 5.42)

	a, _ := New(repo, server.URL, 2, 10)
	a.report()

	expected := []string{
		"/update/gauge/Alloc/5.42",
		"/update/counter/PollCount/0",
	}

	for _, path := range expected {
		if !paths[path] {
			t.Fatalf("expected request to %s", path)
		}
	}
}
