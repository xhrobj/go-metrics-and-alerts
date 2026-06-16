package handler_test

// handler_test.go содержит тесты для актуального JSON API.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
)

// 200 OK для валидного POST /update с типом gauge
func TestHandler_UpdateJSON_Gauge_OK(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	body := `{
		"id": "Alloc",
		"type": "gauge",
		"value": 5.11
	}`

	rq := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateJSON(rr, rq)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	got, ok := repo.gauges["Alloc"]
	require.True(t, ok)

	want := 5.11
	require.Equal(t, want, got)

}

// UpdateJSON передаёт контекст HTTP-запроса в сервис и хранилище.
func TestHandler_UpdateJSON_PassesRequestContext(t *testing.T) {
	type contextKey struct{}

	key := contextKey{}
	wantContextValue := "request-context"

	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	body := `{
		"id": "Alloc",
		"type": "gauge",
		"value": 5.11
	}`

	rq := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rq = rq.WithContext(context.WithValue(rq.Context(), key, wantContextValue))
	rs := httptest.NewRecorder()

	h.UpdateJSON(rs, rq)

	require.Equal(t, http.StatusOK, rs.Code)
	require.NotNil(t, repo.gotContext)

	gotContextValue, _ := repo.gotContext.Value(key).(string)
	require.Equal(t, wantContextValue, gotContextValue)
}

// 200 OK для валидного POST /update с типом counter
func TestHandler_UpdateJSON_Counter_OK(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	body := `{
		"id": "PollCount",
		"type": "counter",
		"delta": 42
	}`

	rq := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateJSON(rr, rq)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	value, ok := repo.counters["PollCount"]
	require.True(t, ok)

	want := int64(42)
	require.Equal(t, want, value)
}

// два POST на одну counter-метрику (для /update) -> счётчик суммируется
func TestHandler_UpdateJSON_Counter_Accumulates_OK(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	body := `{
		"id": "PollCount",
		"type": "counter",
		"delta": 42
	}`

	rq1 := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	h.UpdateJSON(rr1, rq1)

	require.Equal(t, http.StatusOK, rr1.Code)

	rq2 := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
	rq2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	h.UpdateJSON(rr2, rq2)

	require.Equal(t, http.StatusOK, rr2.Code)

	assert.Equal(t, "application/json", rr2.Header().Get("Content-Type"))

	got, ok := repo.counters["PollCount"]
	require.True(t, ok)

	want := int64(84)
	require.Equal(t, want, got)
}

// ассорти ошибок для POST /update (таблица кейсов)
func TestHandler_UpdateJSON_StatusCodes(t *testing.T) {
	srv := service.NewMetricsService(newMockServerStorage())
	h := handler.New(srv, nil)

	tests := []struct {
		name        string
		method      string
		body        string
		contentType string
		wantStatus  int
	}{
		// 400 Bad Request -> если Content-Type не application/json:

		{
			name:   "bad content type",
			method: http.MethodPost,
			body: `{
				"id": "Alloc",
				"type": "gauge",
				"value": 1
			}`,
			contentType: "text/plain",
			wantStatus:  http.StatusBadRequest,
		},

		// 400 Bad Request -> если тип метрики неизвестен:

		{
			name:   "unknown type",
			method: http.MethodPost,
			body: `{
				"id": "Alloc",
				"type": "unknown",
				"value": 1
			}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},

		// 400 Bad Request -> если значение не парсится (float/int):

		{
			name:   "bad gauge value",
			method: http.MethodPost,
			body: `{
				"id": "Alloc",
				"type": "gauge",
				"value": "abc"
			}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},

		{
			name:   "bad counter value",
			method: http.MethodPost,
			body: `{
				"id": "PollCount",
				"type": "counter",
				"delta": 1.2
			}`,
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
		},

		// 404 Not Found -> если пустое имя метрики (кейс из требований):

		{
			name:   "empty name",
			method: http.MethodPost,
			body: `{
				"id": "",
				"type": "gauge",
				"value": 1
			}`,
			contentType: "application/json",
			wantStatus:  http.StatusNotFound,
		},

		// 405 Method Not Allowed -> если метод не POST:

		{
			name:   "method not allowed",
			method: http.MethodGet,
			body: `{
				"id": "Alloc",
				"type": "gauge",
				"value": 1
			}`,
			contentType: "application/json",
			wantStatus:  http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rq := httptest.NewRequest(tt.method, "/update", strings.NewReader(tt.body))
			if tt.contentType != "" {
				rq.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()
			h.UpdateJSON(rr, rq)

			assert.Equal(t, tt.wantStatus, rr.Code, "case %q", tt.name)
		})
	}
}

// 200 OK для валидного POST /updates с набором метрик
func TestHandler_UpdatesJSON_OK(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	value := 5.11
	delta := int64(42)

	metrics := []model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		},
		{
			ID:    "PollCount",
			MType: model.Counter,
			Delta: &delta,
		},
	}

	body, err := json.Marshal(metrics)
	require.NoError(t, err)

	rq := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdatesJSON(rr, rq)

	require.Equal(t, http.StatusOK, rr.Code)

	gotGauge, err := srv.GetGauge(context.Background(), "Alloc")
	require.NoError(t, err)

	wantGauge := value
	require.Equal(t, wantGauge, gotGauge)

	gotCounter, err := srv.GetCounter(context.Background(), "PollCount")
	require.NoError(t, err)

	wantCounter := delta
	require.Equal(t, wantCounter, gotCounter)
}

// UpdatesJSON проверяет метод, Content-Type и корректность JSON-запроса.
func TestHandler_UpdatesJSON_RequestValidation(t *testing.T) {
	srv := service.NewMetricsService(newMockServerStorage())
	h := handler.New(srv, nil)

	tests := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantStatus  int
	}{
		{
			name:        "method not allowed",
			method:      http.MethodGet,
			contentType: "application/json",
			body:        `[]`,
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:       "missing content type",
			method:     http.MethodPost,
			body:       `[]`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "bad content type",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        `[]`,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "bad JSON",
			method:      http.MethodPost,
			contentType: "application/json",
			body:        `{"broken":`,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rq := httptest.NewRequest(tt.method, "/updates", strings.NewReader(tt.body))
			if tt.contentType != "" {
				rq.Header.Set("Content-Type", tt.contentType)
			}

			rs := httptest.NewRecorder()
			h.UpdatesJSON(rs, rq)

			require.Equal(t, tt.wantStatus, rs.Code)
		})
	}
}

// 400 Bad Request для POST /updates с невалидной метрикой
func TestHandler_UpdatesJSON_InvalidMetric(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	metrics := []model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
		},
	}

	body, err := json.Marshal(metrics)
	require.NoError(t, err)

	rq := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdatesJSON(rr, rq)

	require.Equal(t, http.StatusBadRequest, rr.Code)
}

// если в батче есть невалидная метрика, ни одна метрика не должна примениться
func TestHandler_UpdatesJSON_InvalidMetric_DoesNotApplyBatch(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	value := 5.11

	metrics := []model.Metrics{
		{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		},
		{
			ID:    "BrokenCounter",
			MType: model.Counter,
		},
	}

	body, err := json.Marshal(metrics)
	require.NoError(t, err)

	rq := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(body))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdatesJSON(rr, rq)

	require.Equal(t, http.StatusBadRequest, rr.Code)

	_, err = srv.GetGauge(context.Background(), "Alloc")
	require.Error(t, err)
}
