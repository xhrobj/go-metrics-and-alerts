package grpcserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingInterceptorLogsSuccessfulRequest(t *testing.T) {
	core, observedLogs := observer.New(zap.InfoLevel)
	log := zap.New(core)
	wantRS := metricspb.UpdateMetricsResponse_builder{}.Build()

	handler := func(context.Context, any) (any, error) {
		return wantRS, nil
	}

	interceptor := LoggingInterceptor(log)
	rs, err := interceptor(
		context.Background(),
		metricspb.UpdateMetricsRequest_builder{}.Build(),
		&grpc.UnaryServerInfo{FullMethod: metricspb.Metrics_UpdateMetrics_FullMethodName},
		handler,
	)

	require.NoError(t, err)
	require.Same(t, wantRS, rs)

	entries := observedLogs.FilterMessage("grpc request completed").All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	require.Equal(t, metricspb.Metrics_UpdateMetrics_FullMethodName, fields["method"])
	require.Equal(t, codes.OK.String(), fields["status"])
	require.Contains(t, fields, "duration")
}

func TestLoggingInterceptorLogsFailedRequest(t *testing.T) {
	core, observedLogs := observer.New(zap.InfoLevel)
	log := zap.New(core)
	wantErr := status.Error(codes.InvalidArgument, "invalid request")

	handler := func(context.Context, any) (any, error) {
		return nil, wantErr
	}

	interceptor := LoggingInterceptor(log)
	rs, err := interceptor(
		context.Background(),
		metricspb.UpdateMetricsRequest_builder{}.Build(),
		&grpc.UnaryServerInfo{FullMethod: metricspb.Metrics_UpdateMetrics_FullMethodName},
		handler,
	)

	require.Nil(t, rs)
	require.ErrorIs(t, err, wantErr)

	entries := observedLogs.FilterMessage("grpc request completed").All()
	require.Len(t, entries, 1)

	fields := entries[0].ContextMap()
	require.Equal(t, metricspb.Metrics_UpdateMetrics_FullMethodName, fields["method"])
	require.Equal(t, codes.InvalidArgument.String(), fields["status"])
	require.Contains(t, fields, "duration")
}

func TestLoggingInterceptorSupportsNilLogger(t *testing.T) {
	wantRS := metricspb.UpdateMetricsResponse_builder{}.Build()
	handler := func(context.Context, any) (any, error) {
		return wantRS, nil
	}

	interceptor := LoggingInterceptor(nil)
	rs, err := interceptor(
		context.Background(),
		metricspb.UpdateMetricsRequest_builder{}.Build(),
		&grpc.UnaryServerInfo{FullMethod: metricspb.Metrics_UpdateMetrics_FullMethodName},
		handler,
	)

	require.NoError(t, err)
	require.Same(t, wantRS, rs)
}
