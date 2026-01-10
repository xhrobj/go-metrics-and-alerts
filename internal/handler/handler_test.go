package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
)

func testRouter(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)
	return r
}

// 200 OK для валидного POST /update/gauge/<name>/<value>
func TestHandler_Update_Gauge_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(h)

	rq := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/5.42", nil)
	rq.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	expected := 5.42
	if got := repo.gauges["Alloc"]; got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "text/plain; charset=utf-8", ct)
	}
}

// 200 OK для валидного POST /update/counter/<name>/<value>
func TestHandler_Update_Counter_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(h)

	rq := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/42", nil)
	rq.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	expected := int64(42)
	if got := repo.counters["PollCount"]; got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "text/plain; charset=utf-8", ct)
	}
}

// два POST на одну counter-метрику -> счётчик суммируется
func TestHandler_Update_Counter_Accumulates_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(h)

	rq := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/42", nil)
	rq.Header.Set("Content-Type", "text/plain")

	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, rq)

	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, rq)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr2.Code)
	}

	expected := int64(84)
	if got := repo.counters["PollCount"]; got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestHandler_Update_StatusCodes(t *testing.T) {
	h := handler.New(newMockServerStorage())
	r := testRouter(h)

	tests := []struct {
		name        string
		method      string
		path        string
		contentType string
		wantStatus  int
	}{
		// 400 Bad Request -> если Content-Type не text/plain:

		{"bad content type", http.MethodPost, "/update/gauge/Alloc/1", "application/json", http.StatusBadRequest},

		// 400 Bad Request -> если тип метрики неизвестен:

		{"unknown type", http.MethodPost, "/update/unknown/Alloc/1", "text/plain", http.StatusBadRequest},

		// 400 Bad Request -> если значение не парсится (float/int):

		{"bad gauge value", http.MethodPost, "/update/gauge/Alloc/abc", "text/plain", http.StatusBadRequest},
		{"bad counter value", http.MethodPost, "/update/counter/PollCount/1.2", "text/plain", http.StatusBadRequest},

		// 404 Not Found -> если путь “не той формы” (мало/много сегментов, пустые сегменты):

		{"malformed path short", http.MethodPost, "/update/gauge/Alloc", "text/plain", http.StatusNotFound},
		{"malformed path long", http.MethodPost, "/update/gauge/Alloc/1/extra", "text/plain", http.StatusNotFound},

		{"empty value", http.MethodPost, "/update/gauge/Alloc/", "text/plain", http.StatusNotFound},
		{"empty all", http.MethodPost, "/update////", "text/plain", http.StatusNotFound},

		// 404 Not Found -> если пустое имя метрики (кейс из требований):

		{"empty name", http.MethodPost, "/update/gauge//1", "text/plain", http.StatusNotFound},

		// 405 Method Not Allowed -> если метод не POST:

		{"method not allowed", http.MethodGet, "/update/gauge/Alloc/1", "text/plain", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rq := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.contentType != "" {
				rq.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, rq)

			if rr.Code != tt.wantStatus {
				t.Errorf("expected %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}
