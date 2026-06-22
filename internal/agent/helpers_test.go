package agent

import (
	"context"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

type senderFunc func(context.Context, []model.Metrics) error

func (f senderFunc) Send(ctx context.Context, metrics []model.Metrics) error {
	return f(ctx, metrics)
}

func newNoopSender() MetricsSender {
	return senderFunc(func(context.Context, []model.Metrics) error {
		return nil
	})
}
