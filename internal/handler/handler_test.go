package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
)

func testRouter(t *testing.T, h *handler.Handler) http.Handler {
	t.Helper()

	r := chi.NewRouter()
	r.Post("/update", h.UpdateJSON)
	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Get("/value/{type}/{name}", h.Value)
	r.Get("/", h.Index)

	return r
}

// 200 OK для валидного POST /update/gauge/<name>/<value>
func TestHandler_Update_Gauge_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(t, h)

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

// 200 OK для валидного POST /update с типом gauge
func TestHandler_UpdateJSON_Gauge_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(t, h)

	body := `{
		"id": "Alloc",
		"type": "gauge",
		"value": 5.42
	}`

	rq := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	expected := 5.42
	if got := repo.gauges["Alloc"]; got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", ct)
	}
}

// 200 OK для валидного POST /update/counter/<name>/<value>
func TestHandler_Update_Counter_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(t, h)

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

// 200 OK для валидного POST /update с типом counter
func TestHandler_UpdateJSON_Counter_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(t, h)

	body := `{
		"id": "PollCount",
		"type": "counter",
		"delta": 42
	}`

	rq := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	expected := int64(42)
	if got := repo.counters["PollCount"]; got != expected {
		t.Fatalf("expected %v, got %v", expected, got)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type %q, got %q", "application/json", ct)
	}
}

// два POST на одну counter-метрику (для /update/counter/...) -> счётчик суммируется
func TestHandler_Update_Counter_Accumulates_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(t, h)

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
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// два POST на одну counter-метрику (для /update) -> счётчик суммируется
func TestHandler_UpdateJSON_Counter_Accumulates_OK(t *testing.T) {
	repo := newMockServerStorage()
	h := handler.New(repo)
	r := testRouter(t, h)

	body := `{
		"id": "PollCount",
		"type": "counter",
		"delta": 42
	}`

	rq1 := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, rq1)

	rq2 := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, rq2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr2.Code)
	}

	expected := int64(84)
	if got := repo.counters["PollCount"]; got != expected {
		t.Errorf("expected %v, got %v", expected, got)
	}
}

// ассорти ошибок для POST /update/... (таблица кейсов)
func TestHandler_Update_StatusCodes(t *testing.T) {
	h := handler.New(newMockServerStorage())
	r := testRouter(t, h)

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
				t.Errorf("%s: expected %d, got %d", tt.name, tt.wantStatus, rr.Code)
			}
		})
	}
}

// ассорти ошибок для POST /update (таблица кейсов)
func TestHandler_UpdateJSON_StatusCodes(t *testing.T) {
	h := handler.New(newMockServerStorage())
	r := testRouter(t, h)

	tests := []struct {
		name        string
		method      string
		body        string
		contentType string
		wantStatus  int
	}{
		// 400 Bad Request -> если Content-Type не application/json:

		{"bad content type", http.MethodPost, `{
			"id": "Alloc",
			"type": "gauge",
			"value": 1
		}`,
			"text/plain", http.StatusBadRequest},

		// 400 Bad Request -> если тип метрики неизвестен:

		{"unknown type", http.MethodPost, `{
			"id": "Alloc",
			"type": "unknown",
			"value": 1
		}`,
			"application/json", http.StatusBadRequest},

		// 400 Bad Request -> если значение не парсится (float/int):

		{"bad gauge value", http.MethodPost, `{
			"id": "Alloc",
			"type": "gauge",
			"value": "abc"
		}`,
			"application/json", http.StatusBadRequest},

		{"bad counter value", http.MethodPost, `{
			"id": "PollCount",
			"type": "counter",
			"delta": 1.2
		}`,
			"application/json", http.StatusBadRequest},

		// 404 Not Found -> если пустое имя метрики (кейс из требований):

		{"empty name", http.MethodPost, `{
			"id": "",
			"type": "gauge",
			"value": 1
		}`,
			"application/json", http.StatusNotFound},

		// 405 Method Not Allowed -> если метод не POST:

		{"method not allowed", http.MethodGet, `{
			"id": "Alloc",
			"type": "gauge",
			"value": 1
		}`,
			"application/json", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rq := httptest.NewRequest(tt.method, "/update", strings.NewReader(tt.body))
			if tt.contentType != "" {
				rq.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, rq)

			if rr.Code != tt.wantStatus {
				t.Errorf("%s: expected %d, got %d", tt.name, tt.wantStatus, rr.Code)
			}
		})
	}
}

func TestHandler_Value(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		seed       func(repo *mockServerStorage)
		wantStatus int
		wantBody   string
		wantCT     string
	}{
		{
			name: "gauge ok",
			path: "/value/gauge/Alloc",
			seed: func(repo *mockServerStorage) {
				repo.gauges["Alloc"] = 5.42
			},
			wantStatus: http.StatusOK,
			wantBody:   "5.42",
			wantCT:     "text/plain; charset=utf-8",
		},
		{
			name: "counter ok",
			path: "/value/counter/PollCount",
			seed: func(repo *mockServerStorage) {
				repo.counters["PollCount"] = 42
			},
			wantStatus: http.StatusOK,
			wantBody:   "42",
			wantCT:     "text/plain; charset=utf-8",
		},
		{
			name:       "unknown type -> 404",
			path:       "/value/unknown/Alloc",
			seed:       func(repo *mockServerStorage) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "metric not found -> 404",
			path:       "/value/gauge/NoSuchMetric",
			seed:       func(repo *mockServerStorage) {},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockServerStorage()
			tt.seed(repo)

			h := handler.New(repo)
			r := testRouter(t, h)

			rq := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, rq)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantStatus == http.StatusOK {
				if body := rr.Body.String(); body != tt.wantBody {
					t.Fatalf("expected body %q, got %q", tt.wantBody, body)
				}
				if ct := rr.Header().Get("Content-Type"); ct != tt.wantCT {
					t.Fatalf("expected Content-Type %q, got %q", tt.wantCT, ct)
				}
			}
		})
	}
}

func TestHandler_Index_OK(t *testing.T) {
	repo := newMockServerStorage()
	repo.gauges["Alloc"] = 5.42
	repo.counters["PollCount"] = 42

	h := handler.New(repo)
	r := testRouter(t, h)

	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("expected Content-Type %q, got %q", "text/html; charset=utf-8", ct)
	}

	body := rr.Body.String()

	if !(strings.Contains(body, "Alloc") && strings.Contains(body, "5.42")) {
		t.Errorf("response body does not contain gauge metric: %s", body)
	}

	if !(strings.Contains(body, "PollCount") && strings.Contains(body, "42")) {
		t.Errorf("response body does not contain counter metric: %s", body)
	}
}
