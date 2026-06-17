package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	handlermocks "github.com/xhrobj/go-metrics-and-alerts/internal/handler/mocks"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/mock/gomock"
)

// ValueJSON проверяет метод, Content-Type и корректность JSON-запроса.
func TestHandler_ValueJSON_RequestValidation(t *testing.T) {
	srv := newMockService(t)
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
			body:        `{"id":"Alloc","type":"gauge"}`,
			wantStatus:  http.StatusMethodNotAllowed,
		},
		{
			name:       "missing content type",
			method:     http.MethodPost,
			body:       `{"id":"Alloc","type":"gauge"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:        "bad content type",
			method:      http.MethodPost,
			contentType: "text/plain",
			body:        `{"id":"Alloc","type":"gauge"}`,
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
			rq := httptest.NewRequest(tt.method, "/value", strings.NewReader(tt.body))
			if tt.contentType != "" {
				rq.Header.Set("Content-Type", tt.contentType)
			}

			rs := httptest.NewRecorder()
			h.ValueJSON(rs, rq)

			require.Equal(t, tt.wantStatus, rs.Code)
		})
	}
}

// ValueJSON возвращает текущее значение метрики в формате JSON
func TestHandler_ValueJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(*handlermocks.MockService)
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
			setup: func(srv *handlermocks.MockService) {
				srv.EXPECT().
					GetGauge(gomock.Any(), "Alloc").
					Return(5.11, nil)
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
			setup: func(srv *handlermocks.MockService) {
				srv.EXPECT().
					GetCounter(gomock.Any(), "PollCount").
					Return(int64(42), nil)
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
			setup:      func(_ *handlermocks.MockService) {},
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
			setup: func(srv *handlermocks.MockService) {
				srv.EXPECT().
					GetGauge(gomock.Any(), "NoSuchMetric").
					Return(float64(0), repository.ErrMetricNotFound)
			},
			wantStatus: http.StatusNotFound,
			wantValue:  0,
			wantDelta:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newMockService(t)
			tt.setup(srv)

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
	srv := newMockService(t)
	srv.EXPECT().
		Snapshot(gomock.Any()).
		Return(
			map[string]float64{"Alloc": 5.11},
			map[string]int64{"PollCount": 42},
			nil,
		)

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
