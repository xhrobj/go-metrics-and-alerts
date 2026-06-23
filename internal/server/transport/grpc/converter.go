package grpcserver

import (
	"fmt"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
)

func metricsFromProto(request *metricspb.UpdateMetricsRequest) ([]model.Metrics, error) {
	protoMetrics := request.GetMetrics()
	metrics := make([]model.Metrics, 0, len(protoMetrics))

	for i, protoMetric := range protoMetrics {
		metric, err := metricFromProto(protoMetric)
		if err != nil {
			return nil, fmt.Errorf("metric[%d]: %w", i, err)
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func metricFromProto(protoMetric *metricspb.Metric) (model.Metrics, error) {
	switch protoMetric.GetType() {
	case metricspb.Metric_GAUGE:
		value := protoMetric.GetValue()

		return model.Metrics{
			ID:    protoMetric.GetId(),
			MType: model.Gauge,
			Value: &value,
		}, nil

	case metricspb.Metric_COUNTER:
		delta := protoMetric.GetDelta()

		return model.Metrics{
			ID:    protoMetric.GetId(),
			MType: model.Counter,
			Delta: &delta,
		}, nil

	default:
		return model.Metrics{}, fmt.Errorf(
			"unknown metric type: %d",
			protoMetric.GetType(),
		)
	}
}
