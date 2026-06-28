package grpctransport

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
)

func TestRequestFromMetrics(t *testing.T) {
	gaugeValue := 5.11
	counterDelta := int64(42)

	rq, err := requestFromMetrics([]model.Metrics{
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
	})
	require.NoError(t, err)

	metrics := rq.GetMetrics()
	require.Len(t, metrics, 2)

	require.Equal(t, "Alloc", metrics[0].GetId())
	require.Equal(t, metricspb.Metric_GAUGE, metrics[0].GetType())
	require.Equal(t, gaugeValue, metrics[0].GetValue())

	require.Equal(t, "PollCount", metrics[1].GetId())
	require.Equal(t, metricspb.Metric_COUNTER, metrics[1].GetType())
	require.Equal(t, counterDelta, metrics[1].GetDelta())
}

func TestRequestFromMetricsAllowsEmptyBatch(t *testing.T) {
	rq, err := requestFromMetrics(nil)
	require.NoError(t, err)
	require.Empty(t, rq.GetMetrics())
}

func TestRequestFromMetricsRejectsInvalidMetric(t *testing.T) {
	gaugeValue := 5.11
	counterDelta := int64(42)

	tests := []struct {
		name      string
		metric    model.Metrics
		wantError string
	}{
		{
			name: "empty ID",
			metric: model.Metrics{
				MType: model.Gauge,
				Value: &gaugeValue,
			},
			wantError: "metric ID must not be empty",
		},
		{
			name: "gauge without value",
			metric: model.Metrics{
				ID:    "Alloc",
				MType: model.Gauge,
			},
			wantError: "nil value",
		},
		{
			name: "counter without delta",
			metric: model.Metrics{
				ID:    "PollCount",
				MType: model.Counter,
			},
			wantError: "nil delta",
		},
		{
			name: "unknown type",
			metric: model.Metrics{
				ID:    "Alloc",
				MType: "unknown",
				Value: &gaugeValue,
				Delta: &counterDelta,
			},
			wantError: `unknown type "unknown"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := requestFromMetrics([]model.Metrics{tt.metric})
			require.ErrorContains(t, err, tt.wantError)
		})
	}
}
