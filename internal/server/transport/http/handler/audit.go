package handler

import (
	"net"
	"net/http"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"go.uber.org/zap"
)

func (h *Handler) notifyAuditMetric(r *http.Request, metricName string) {
	if h.auditor == nil || metricName == "" {
		return
	}

	h.notifyAudit(r, []string{metricName})
}

func (h *Handler) notifyAuditMetrics(r *http.Request, metrics []model.Metrics) {
	if h.auditor == nil || len(metrics) == 0 {
		return
	}

	h.notifyAudit(r, metricIDs(metrics))
}

func (h *Handler) notifyAudit(r *http.Request, metrics []string) {
	if h.auditor == nil || len(metrics) == 0 {
		return
	}

	event := audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: clientIP(r),
	}

	if err := h.auditor.Notify(r.Context(), event); err != nil && h.log != nil {
		h.log.Error("audit failed", zap.Error(err))
	}
}

func clientIP(r *http.Request) string {
	if realIP := r.Header.Get(protocol.HeaderRealIP); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func metricIDs(metrics []model.Metrics) []string {
	ids := make([]string, 0, len(metrics))

	for _, metric := range metrics {
		if metric.ID != "" {
			ids = append(ids, metric.ID)
		}
	}

	return ids
}
