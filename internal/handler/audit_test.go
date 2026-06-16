package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
	"go.uber.org/zap"
)

type mockAuditor struct {
	events []audit.Event
	err    error
}

func (m *mockAuditor) Notify(
	_ context.Context,
	event audit.Event,
) error {
	m.events = append(m.events, event)

	return m.err
}

func TestHandler_UpdateJSON_NotifiesAuditor(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	auditor := &mockAuditor{
		err: errors.New("audit failed"),
	}
	h.EnableAudit(auditor, zap.NewNop())

	body := `{
		"id": "Alloc",
		"type": "gauge",
		"value": 5.11
	}`

	rq := httptest.NewRequest(
		http.MethodPost,
		"/update",
		bytes.NewBufferString(body),
	)
	rq.Header.Set("Content-Type", "application/json")
	rq.RemoteAddr = "203.0.113.10:4321"

	rs := httptest.NewRecorder()

	h.UpdateJSON(rs, rq)

	require.Equal(t, http.StatusOK, rs.Code)
	require.Len(t, auditor.events, 1)

	event := auditor.events[0]

	require.NotZero(t, event.TS)
	require.Equal(t, []string{"Alloc"}, event.Metrics)
	require.Equal(t, "203.0.113.10", event.IPAddress)
}

func TestHandler_UpdatesJSON_NotifiesAuditor(t *testing.T) {
	repo := newMockServerStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv, nil)

	auditor := &mockAuditor{}
	h.EnableAudit(auditor, zap.NewNop())

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

	rq := httptest.NewRequest(
		http.MethodPost,
		"/updates",
		bytes.NewReader(body),
	)
	rq.Header.Set("Content-Type", "application/json")

	// NOTE: без host:port — покрываем fallback-ветку clientIP
	rq.RemoteAddr = "test-client"

	rs := httptest.NewRecorder()

	h.UpdatesJSON(rs, rq)

	require.Equal(t, http.StatusOK, rs.Code)
	require.Len(t, auditor.events, 1)

	event := auditor.events[0]

	require.NotZero(t, event.TS)
	require.Equal(t, []string{"Alloc", "PollCount"}, event.Metrics)
	require.Equal(t, "test-client", event.IPAddress)
}
