package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceStub struct {
	called     bool
	gotMetrics []model.Metrics
	err        error
}

func (s *serviceStub) UpdateMetrics(_ context.Context, metrics []model.Metrics) error {
	s.called = true
	s.gotMetrics = metrics

	return s.err
}

func TestServerUpdateMetrics(t *testing.T) {
	service := &serviceStub{}
	server := New(service)

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

	rs, err := server.UpdateMetrics(context.Background(), rq)

	require.NoError(t, err)
	require.NotNil(t, rs)
	require.True(t, service.called)

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

	require.Equal(t, want, service.gotMetrics)
}

func TestServerUpdateMetricsMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{
			name:     "canceled",
			err:      context.Canceled,
			wantCode: codes.Canceled,
		},
		{
			name:     "deadline exceeded",
			err:      context.DeadlineExceeded,
			wantCode: codes.DeadlineExceeded,
		},
		{
			name:     "internal error",
			err:      errors.New("repository unavailable"),
			wantCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &serviceStub{
				err: tt.err,
			}
			server := New(service)

			rq := metricspb.UpdateMetricsRequest_builder{}.Build()

			rs, err := server.UpdateMetrics(context.Background(), rq)

			require.Nil(t, rs)
			require.Equal(t, tt.wantCode, status.Code(err))
			require.True(t, service.called)
		})
	}
}

func TestServerUpdateMetricsDoesNotExposeInternalError(t *testing.T) {
	service := &serviceStub{
		err: errors.New("postgres password leaked here"),
	}
	server := New(service)

	rq := metricspb.UpdateMetricsRequest_builder{}.Build()

	rs, err := server.UpdateMetrics(context.Background(), rq)

	require.Nil(t, rs)
	require.Equal(t, codes.Internal, status.Code(err))
	require.Equal(t, "update metrics failed", status.Convert(err).Message())
}
