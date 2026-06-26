package grpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingInterceptorUsesLevelByStatus(t *testing.T) {
	tests := []struct {
		name       string
		handlerErr error
		wantCode   codes.Code
		wantLevel  zapcore.Level
	}{
		{
			name:      "successful request",
			wantCode:  codes.OK,
			wantLevel: zap.DebugLevel,
		},
		{
			name:       "client error",
			handlerErr: status.Error(codes.InvalidArgument, "invalid request"),
			wantCode:   codes.InvalidArgument,
			wantLevel:  zap.DebugLevel,
		},
		{
			name:       "unknown error",
			handlerErr: errors.New("unexpected failure"),
			wantCode:   codes.Unknown,
			wantLevel:  zap.ErrorLevel,
		},
		{
			name:       "unimplemented",
			handlerErr: status.Error(codes.Unimplemented, "not implemented"),
			wantCode:   codes.Unimplemented,
			wantLevel:  zap.ErrorLevel,
		},
		{
			name:       "internal error",
			handlerErr: status.Error(codes.Internal, "update failed"),
			wantCode:   codes.Internal,
			wantLevel:  zap.ErrorLevel,
		},
		{
			name:       "unavailable",
			handlerErr: status.Error(codes.Unavailable, "service unavailable"),
			wantCode:   codes.Unavailable,
			wantLevel:  zap.ErrorLevel,
		},
		{
			name:       "data loss",
			handlerErr: status.Error(codes.DataLoss, "data corrupted"),
			wantCode:   codes.DataLoss,
			wantLevel:  zap.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, observedLogs := observer.New(zap.DebugLevel)
			log := zap.New(core)
			wantRS := metricspb.UpdateMetricsResponse_builder{}.Build()

			handler := func(context.Context, any) (any, error) {
				if tt.handlerErr != nil {
					return nil, tt.handlerErr
				}

				return wantRS, nil
			}

			interceptor := LoggingInterceptor(log)
			rs, err := interceptor(
				context.Background(),
				metricspb.UpdateMetricsRequest_builder{}.Build(),
				&grpc.UnaryServerInfo{
					FullMethod: metricspb.Metrics_UpdateMetrics_FullMethodName,
				},
				handler,
			)

			if tt.handlerErr != nil {
				require.Nil(t, rs)
				require.ErrorIs(t, err, tt.handlerErr)
			} else {
				require.NoError(t, err)
				require.Same(t, wantRS, rs)
			}

			entries := observedLogs.FilterMessage("grpc request completed").All()
			require.Len(t, entries, 1)
			require.Equal(t, tt.wantLevel, entries[0].Level)

			fields := entries[0].ContextMap()
			require.Equal(
				t,
				metricspb.Metrics_UpdateMetrics_FullMethodName,
				fields["method"],
			)
			require.Equal(t, tt.wantCode.String(), fields["status"])
			require.Contains(t, fields, "duration")
		})
	}
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
		&grpc.UnaryServerInfo{
			FullMethod: metricspb.Metrics_UpdateMetrics_FullMethodName,
		},
		handler,
	)

	require.NoError(t, err)
	require.Same(t, wantRS, rs)
}
