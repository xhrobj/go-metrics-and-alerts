package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type serviceStub struct {
	calls      int
	gotContext context.Context
	gotMetrics []model.Metrics
	err        error
}

func (s *serviceStub) UpdateMetrics(ctx context.Context, metrics []model.Metrics) error {
	s.calls++
	s.gotContext = ctx
	s.gotMetrics = metrics

	return s.err
}

type auditorStub struct {
	calls      int
	gotContext context.Context
	events     []audit.Event
	err        error
}

func (a *auditorStub) Notify(ctx context.Context, event audit.Event) error {
	a.calls++
	a.gotContext = ctx
	a.events = append(a.events, event)

	return a.err
}

func TestServerUpdateMetrics(t *testing.T) {
	srv := &serviceStub{}
	grpcServer := New(srv, zap.NewNop())

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

	ctx := context.Background()
	rs, err := grpcServer.UpdateMetrics(ctx, rq)

	require.NoError(t, err)
	require.NotNil(t, rs)
	require.Equal(t, 1, srv.calls, "got calls %d, want 1", srv.calls)
	require.Equal(t, ctx, srv.gotContext)

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

	require.Equal(t, want, srv.gotMetrics)
}

func TestServerUpdateMetricsRejectsInvalidRequest(t *testing.T) {
	srv := &serviceStub{}
	grpcServer := New(srv, zap.NewNop())

	rs, err := grpcServer.UpdateMetrics(context.Background(), nil)

	require.Nil(t, rs)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Equal(t, "request is nil", status.Convert(err).Message())
	require.Zero(t, srv.calls)
}

func TestServerUpdateMetricsMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantCode    codes.Code
		wantMessage string
	}{
		{
			name:        "canceled",
			err:         context.Canceled,
			wantCode:    codes.Canceled,
			wantMessage: context.Canceled.Error(),
		},
		{
			name:        "wrapped canceled",
			err:         fmt.Errorf("update metrics: %w", context.Canceled),
			wantCode:    codes.Canceled,
			wantMessage: context.Canceled.Error(),
		},
		{
			name:        "deadline exceeded",
			err:         context.DeadlineExceeded,
			wantCode:    codes.DeadlineExceeded,
			wantMessage: context.DeadlineExceeded.Error(),
		},
		{
			name:        "wrapped deadline exceeded",
			err:         fmt.Errorf("update metrics: %w", context.DeadlineExceeded),
			wantCode:    codes.DeadlineExceeded,
			wantMessage: context.DeadlineExceeded.Error(),
		},
		{
			name:        "internal error",
			err:         errors.New("repository unavailable"),
			wantCode:    codes.Internal,
			wantMessage: "update metrics failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &serviceStub{err: tt.err}
			grpcServer := New(srv, zap.NewNop())
			rq := metricspb.UpdateMetricsRequest_builder{}.Build()

			rs, err := grpcServer.UpdateMetrics(context.Background(), rq)

			require.Nil(t, rs)
			require.Equal(t, tt.wantCode, status.Code(err))
			require.Equal(t, tt.wantMessage, status.Convert(err).Message())
			require.Equal(t, 1, srv.calls, "got calls %d, want 1", srv.calls)
		})
	}
}

func TestServerUpdateMetricsLogsServiceError(t *testing.T) {
	wantErr := errors.New("repository unavailable")
	srv := &serviceStub{err: wantErr}

	core, observedLogs := observer.New(zap.ErrorLevel)
	grpcServer := New(srv, zap.New(core))
	rq := metricspb.UpdateMetricsRequest_builder{}.Build()

	rs, err := grpcServer.UpdateMetrics(context.Background(), rq)

	require.Nil(t, rs)
	require.Equal(t, codes.Internal, status.Code(err))

	entries := observedLogs.FilterMessage("update metrics failed").All()
	require.Len(t, entries, 1)
	require.Equal(t, wantErr.Error(), entries[0].ContextMap()["error"])
}

func TestServerUpdateMetricsNotifiesAuditor(t *testing.T) {
	srv := &serviceStub{}
	auditor := &auditorStub{}
	grpcServer := New(srv, zap.NewNop())
	grpcServer.EnableAudit(auditor)

	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(protocol.MetadataRealIP, "203.0.113.10"),
	)
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

	rs, err := grpcServer.UpdateMetrics(ctx, rq)

	require.NoError(t, err)
	require.NotNil(t, rs)
	require.Equal(t, 1, auditor.calls, "got calls %d, want 1", auditor.calls)
	require.Equal(t, ctx, auditor.gotContext)
	require.Len(t, auditor.events, 1)

	event := auditor.events[0]
	require.NotZero(t, event.TS)
	require.Equal(t, []string{"Alloc", "PollCount"}, event.Metrics)
	require.Equal(t, "203.0.113.10", event.IPAddress)
}

func TestServerUpdateMetricsIgnoresAuditorError(t *testing.T) {
	srv := &serviceStub{}
	auditor := &auditorStub{err: errors.New("audit failed")}
	grpcServer := New(srv, zap.NewNop())
	grpcServer.EnableAudit(auditor)

	rq := metricspb.UpdateMetricsRequest_builder{
		Metrics: []*metricspb.Metric{
			metricspb.Metric_builder{
				Id:    "Alloc",
				Type:  metricspb.Metric_GAUGE,
				Value: 5.11,
			}.Build(),
		},
	}.Build()

	rs, err := grpcServer.UpdateMetrics(context.Background(), rq)

	require.NoError(t, err)
	require.NotNil(t, rs)
	require.Equal(t, 1, auditor.calls, "got calls %d, want 1", auditor.calls)
}

func TestServerUpdateMetricsDoesNotNotifyAuditor(t *testing.T) {
	tests := []struct {
		name    string
		srvErr  error
		rq      *metricspb.UpdateMetricsRequest
		wantErr bool
	}{
		{
			name:   "service error",
			srvErr: errors.New("repository unavailable"),
			rq: metricspb.UpdateMetricsRequest_builder{
				Metrics: []*metricspb.Metric{
					metricspb.Metric_builder{
						Id:    "Alloc",
						Type:  metricspb.Metric_GAUGE,
						Value: 5.11,
					}.Build(),
				},
			}.Build(),
			wantErr: true,
		},
		{
			name: "empty batch",
			rq:   metricspb.UpdateMetricsRequest_builder{}.Build(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &serviceStub{err: tt.srvErr}
			auditor := &auditorStub{}
			grpcServer := New(srv, zap.NewNop())
			grpcServer.EnableAudit(auditor)

			rs, err := grpcServer.UpdateMetrics(context.Background(), tt.rq)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, rs)
			} else {
				require.NoError(t, err)
				require.NotNil(t, rs)
			}
			require.Zero(t, auditor.calls)
		})
	}
}
