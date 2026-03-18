package handler_test

// handler_test.go содержит тесты для актуального JSON API.

import (
	"bytes"
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

	gotGauge, err := srv.GetGauge("Alloc")
	require.NoError(t, err)

	wantGauge := value
	require.Equal(t, wantGauge, gotGauge)

	gotCounter, err := srv.GetCounter("PollCount")
	require.NoError(t, err)

	wantCounter := delta
	require.Equal(t, wantCounter, gotCounter)
}

// 400 Bad Request для POST /updates с битым JSON
func TestHandler_UpdatesJSON_BadJSON(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	rq := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(`{"broken":`))
	rq.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdatesJSON(rr, rq)

	require.Equal(t, http.StatusBadRequest, rr.Code)
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

	_, err = srv.GetGauge("Alloc")
	require.Error(t, err)
}

// ValueJSON возвращает текущее значение метрики в формате JSON
func TestHandler_ValueJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		seed       func(repo *mockServerStorage)
		wantStatus int
		wantCT     string
		wantValue  float64
		wantDelta  int64
	}{
		{
			name: "gauge ok",
			body: `{
				"id": "Alloc",
				"type": "gauge"
			}`,
			seed: func(repo *mockServerStorage) {
				repo.gauges["Alloc"] = 5.11
			},
			wantStatus: http.StatusOK,
			wantCT:     "application/json",
			wantValue:  5.11,
			wantDelta:  0,
		},
		{
			name: "counter ok",
			body: `{
				"id": "PollCount",
				"type": "counter"
			}`,
			seed: func(repo *mockServerStorage) {
				repo.counters["PollCount"] = 42
			},
			wantStatus: http.StatusOK,
			wantCT:     "application/json",
			wantValue:  0,
			wantDelta:  42,
		},
		{
			name: "unknown type -> 404",
			body: `{
				"id": "Alloc",
				"type": "unknown"
			}`,
			seed:       func(repo *mockServerStorage) {},
			wantStatus: http.StatusNotFound,
			wantValue:  0,
			wantDelta:  0,
		},
		{
			name: "metric not found -> 404",
			body: `{
				"id": "NoSuchMetric",
				"type": "gauge"
			}`,
			seed:       func(repo *mockServerStorage) {},
			wantStatus: http.StatusNotFound,
			wantValue:  0,
			wantDelta:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockServerStorage()
			tt.seed(repo)

			srv := service.NewMetricsService(repo)
			h := handler.New(srv, nil)

			rq := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.body))
			rq.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			h.ValueJSON(rr, rq)

			require.Equal(t, tt.wantStatus, rr.Code)

			if tt.wantStatus == http.StatusOK {
				assert.Equal(t, tt.wantCT, rr.Header().Get("Content-Type"))

				var got model.Metrics
				require.NoError(t, json.NewDecoder(rr.Body).Decode(&got))

				require.False(t, got.Value == nil && got.Delta == nil)
				require.False(t, got.Value != nil && got.Delta != nil)

				if got.Value != nil {
					require.Equal(t, tt.wantValue, *got.Value)
				}

				if got.Delta != nil {
					require.Equal(t, tt.wantDelta, *got.Delta)
				}
			}
		})
	}
}

// Index возвращает HTML-страницу со списком метрик, хранящихся в репозитории
func TestHandler_Index_HTML_OK(t *testing.T) {
	repo := newMockServerStorage()
	repo.gauges["Alloc"] = 5.11
	repo.counters["PollCount"] = 42

	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.Index(rr, rq)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "text/html; charset=utf-8", rr.Header().Get("Content-Type"))

	body := rr.Body.String()

	assert.Contains(t, body, "Alloc")
	assert.Contains(t, body, "5.11")

	assert.Contains(t, body, "PollCount")
	assert.Contains(t, body, "42")
}
