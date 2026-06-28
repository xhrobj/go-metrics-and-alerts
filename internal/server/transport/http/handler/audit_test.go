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
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler"
	"go.uber.org/mock/gomock"
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
	srv := newMockService(t)
	srv.EXPECT().
		UpdateGauge(gomock.Any(), "Alloc", 5.11).
		Return(nil)

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
	rq.Header.Set(protocol.HeaderRealIP, "198.51.100.42")
	rq.RemoteAddr = "203.0.113.10:4321"

	rs := httptest.NewRecorder()

	h.UpdateJSON(rs, rq)

	require.Equal(t, http.StatusOK, rs.Code)
	require.Len(t, auditor.events, 1)

	event := auditor.events[0]

	require.NotZero(t, event.TS)
	require.Equal(t, []string{"Alloc"}, event.Metrics)
	require.Equal(t, "198.51.100.42", event.IPAddress)
}

func TestHandler_UpdatesJSON_NotifiesAuditor(t *testing.T) {
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

	srv := newMockService(t)
	srv.EXPECT().
		UpdateMetrics(gomock.Any(), gomock.Eq(metrics)).
		Return(nil)

	h := handler.New(srv, nil)

	auditor := &mockAuditor{}
	h.EnableAudit(auditor, zap.NewNop())

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
