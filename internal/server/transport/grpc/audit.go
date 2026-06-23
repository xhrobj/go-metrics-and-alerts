package grpcserver

import (
	"context"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"go.uber.org/zap"
)

func (s *Server) notifyAudit(ctx context.Context, metrics []model.Metrics) {
	if s.auditor == nil || len(metrics) == 0 {
		return
	}

	ids := metricIDs(metrics)
	if len(ids) == 0 {
		return
	}

	ip, _ := realIPFromContext(ctx)

	event := audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   ids,
		IPAddress: ip,
	}

	if err := s.auditor.Notify(ctx, event); err != nil {
		s.log.Error("audit failed", zap.Error(err))
	}
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
