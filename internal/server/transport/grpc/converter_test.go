package grpcserver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
)

func TestMetricsFromProto(t *testing.T) {
	t.Run("converts batch", func(t *testing.T) {
		request := metricspb.UpdateMetricsRequest_builder{
			Metrics: []*metricspb.Metric{
				metricspb.Metric_builder{
					Id:    "Alloc",
					Type:  metricspb.Metric_GAUGE,
					Value: 5.11,
				}.Build(),
				metricspb.Metric_builder{
					Id:    "PollCount",
					Type:  metricspb.Metric_COUNTER,
					Delta: 42,
				}.Build(),
			},
		}.Build()

		got, err := metricsFromProto(request)
		require.NoError(t, err)

		gaugeValue := 5.11
		counterDelta := int64(42)

		want := []model.Metrics{
			{
				ID:    "Alloc",
				MType: model.Gauge,
				Value: &gaugeValue,
			},
			{
				ID:    "PollCount",
				MType: model.Counter,
				Delta: &counterDelta,
			},
		}

		require.Equal(t, want, got)
	})

	t.Run("accepts empty batch", func(t *testing.T) {
		request := metricspb.UpdateMetricsRequest_builder{}.Build()

		got, err := metricsFromProto(request)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}
