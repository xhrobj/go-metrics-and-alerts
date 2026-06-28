package grpcserver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
)

func TestMetricsFromProto(t *testing.T) {
	t.Run("converts batch", func(t *testing.T) {
		rq := metricspb.UpdateMetricsRequest_builder{
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

		got, err := metricsFromProto(rq)
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

	t.Run("preserves zero values", func(t *testing.T) {
		rq := metricspb.UpdateMetricsRequest_builder{
			Metrics: []*metricspb.Metric{
				metricspb.Metric_builder{
					Id:   "ZeroGauge",
					Type: metricspb.Metric_GAUGE,
				}.Build(),
				metricspb.Metric_builder{
					Id:   "ZeroCounter",
					Type: metricspb.Metric_COUNTER,
				}.Build(),
			},
		}.Build()

		got, err := metricsFromProto(rq)
		require.NoError(t, err)
		require.Len(t, got, 2)
		require.NotNil(t, got[0].Value)
		require.Equal(t, 0.0, *got[0].Value)
		require.NotNil(t, got[1].Delta)
		require.Equal(t, int64(0), *got[1].Delta)
	})

	t.Run("accepts empty batch", func(t *testing.T) {
		rq := metricspb.UpdateMetricsRequest_builder{}.Build()

		got, err := metricsFromProto(rq)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestMetricsFromProtoRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		rq        *metricspb.UpdateMetricsRequest
		wantError string
	}{
		{
			name:      "nil request",
			rq:        nil,
			wantError: "request is nil",
		},
		{
			name: "nil metric",
			rq: metricspb.UpdateMetricsRequest_builder{
				Metrics: []*metricspb.Metric{nil},
			}.Build(),
			wantError: "metric[0]: metric is nil",
		},
		{
			name: "empty metric id",
			rq: metricspb.UpdateMetricsRequest_builder{
				Metrics: []*metricspb.Metric{
					metricspb.Metric_builder{
						Type:  metricspb.Metric_GAUGE,
						Value: 5.11,
					}.Build(),
				},
			}.Build(),
			wantError: "metric[0]: metric id is empty",
		},
		{
			name: "unknown metric type",
			rq: metricspb.UpdateMetricsRequest_builder{
				Metrics: []*metricspb.Metric{
					metricspb.Metric_builder{
						Id:   "Unknown",
						Type: metricspb.Metric_MType(69),
					}.Build(),
				},
			}.Build(),
			wantError: "metric[0]: unknown metric type: 69",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := metricsFromProto(tt.rq)

			require.ErrorContains(t, err, tt.wantError)
			require.Nil(t, got)
		})
	}
}
