package handler_test

// handler_legacy_test.go содержит тесты для "устаревших" path-based эндпоинтов.
//
// Неиспользуемые в текущей реализации Агентом эндпоинты:
//
//   POST /update/{type}/{name}/{value}
//   GET  /value/{type}/{name}
//
// В текущей реализации Агент работает только с JSON API.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler"
	handlermocks "github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler/mocks"
	"go.uber.org/mock/gomock"
)

// testRouter собирает HTTP-роутер для тестов совместимого path-based API
func testRouter(t *testing.T, h *handler.Handler) http.Handler {
	t.Helper()

	r := chi.NewRouter()

	r.Use(middleware.StripSlashes)

	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Get("/value/{type}/{name}", h.Value)

	return r
}

// 200 OK для валидного POST /update/gauge/<name>/<value>
func TestHandler_Update_Gauge_OK(t *testing.T) {
	srv := newMockService(t)
	srv.EXPECT().
		UpdateGauge(gomock.Any(), "Alloc", 5.11).
		Return(nil)

	h := handler.New(srv, nil)
	r := testRouter(t, h)

	rq := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/5.11", nil)
	rq.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "text/plain; charset=utf-8", ct)
	}
}

// 200 OK для валидного POST /update/counter/<name>/<value>
func TestHandler_Update_Counter_OK(t *testing.T) {
	srv := newMockService(t)
	srv.EXPECT().
		UpdateCounter(gomock.Any(), "PollCount", int64(42)).
		Return(int64(42), nil)

	h := handler.New(srv, nil)
	r := testRouter(t, h)

	rq := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/42", nil)
	rq.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, rq)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rr.Code)
	}

	ct := rr.Header().Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("expected Content-Type %q, got %q", "text/plain; charset=utf-8", ct)
	}
}

// два POST на одну counter-метрику (для /update/counter/...) -> счётчик суммируется
func TestHandler_Update_Counter_Accumulates_OK(t *testing.T) {
	srv := newMockService(t)

	var total int64
	srv.EXPECT().
		UpdateCounter(gomock.Any(), "PollCount", int64(42)).
		DoAndReturn(func(_ context.Context, _ string, delta int64) (int64, error) {
			total += delta
			return total, nil
		}).
		Times(2)

	h := handler.New(srv, nil)
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

	if total != 84 {
		t.Errorf("expected %v, got %v", int64(84), total)
	}
}

// ассорти ошибок для POST /update/... (таблица кейсов)
func TestHandler_Update_StatusCodes(t *testing.T) {
	srv := newMockService(t)
	h := handler.New(srv, nil)
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

// GET /value/{type}/{name} возвращает текущее значение метрики в текстовом виде
func TestHandler_Value(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		setup      func(*handlermocks.MockService)
		wantStatus int
		wantBody   string
		wantCT     string
	}{
		{
			name: "gauge ok",
			path: "/value/gauge/Alloc",
			setup: func(srv *handlermocks.MockService) {
				srv.EXPECT().
					GetGauge(gomock.Any(), "Alloc").
					Return(5.11, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "5.11",
			wantCT:     "text/plain; charset=utf-8",
		},
		{
			name: "counter ok",
			path: "/value/counter/PollCount",
			setup: func(srv *handlermocks.MockService) {
				srv.EXPECT().
					GetCounter(gomock.Any(), "PollCount").
					Return(int64(42), nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "42",
			wantCT:     "text/plain; charset=utf-8",
		},
		{
			name:       "unknown type -> 404",
			path:       "/value/unknown/Alloc",
			setup:      func(_ *handlermocks.MockService) {},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "metric not found -> 404",
			path: "/value/gauge/NoSuchMetric",
			setup: func(srv *handlermocks.MockService) {
				srv.EXPECT().
					GetGauge(gomock.Any(), "NoSuchMetric").
					Return(float64(0), repository.ErrMetricNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newMockService(t)
			tt.setup(srv)

			h := handler.New(srv, nil)
			r := testRouter(t, h)

			rq := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, rq)

			if rr.Code != tt.wantStatus {
				t.Fatalf("%s: expected %d, got %d", tt.name, tt.wantStatus, rr.Code)
			}

			if tt.wantStatus == http.StatusOK {
				if body := rr.Body.String(); body != tt.wantBody {
					t.Fatalf("%s: expected body %q, got %q", tt.name, tt.wantBody, body)
				}
				if ct := rr.Header().Get("Content-Type"); ct != tt.wantCT {
					t.Errorf("%s: expected Content-Type %q, got %q", tt.name, tt.wantCT, ct)
				}
			}
		})
	}
}
