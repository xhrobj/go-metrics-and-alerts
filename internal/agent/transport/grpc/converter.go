package grpctransport

import (
	"fmt"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
)

func requestFromMetrics(metrics []model.Metrics) (*metricspb.UpdateMetricsRequest, error) {
	protoMetrics := make([]*metricspb.Metric, 0, len(metrics))

	for i, metric := range metrics {
		converted, err := metricToProto(metric)
		if err != nil {
			return nil, fmt.Errorf("convert metric %d: %w", i, err)
		}

		protoMetrics = append(protoMetrics, converted)
	}

	return metricspb.UpdateMetricsRequest_builder{
		Metrics: protoMetrics,
	}.Build(), nil
}

func metricToProto(metric model.Metrics) (*metricspb.Metric, error) {
	if metric.ID == "" {
		return nil, fmt.Errorf("metric ID must not be empty")
	}

	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return nil, fmt.Errorf("gauge metric %q has nil value", metric.ID)
		}

		return metricspb.Metric_builder{
			Id:    metric.ID,
			Type:  metricspb.Metric_GAUGE,
			Value: *metric.Value,
		}.Build(), nil

	case model.Counter:
		if metric.Delta == nil {
			return nil, fmt.Errorf("counter metric %q has nil delta", metric.ID)
		}

		return metricspb.Metric_builder{
			Id:    metric.ID,
			Type:  metricspb.Metric_COUNTER,
			Delta: *metric.Delta,
		}.Build(), nil

	default:
		return nil, fmt.Errorf(
			"metric %q has unknown type %q",
			metric.ID,
			metric.MType,
		)
	}
}
