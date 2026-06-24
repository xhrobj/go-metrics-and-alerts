// Package grpctransport реализует отправку метрик Агентом через gRPC.
package grpctransport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/service"
	agenttransport "github.com/xhrobj/go-metrics-and-alerts/internal/agent/transport"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var defaultRetryDelays = []time.Duration{
	time.Second,
	3 * time.Second,
	5 * time.Second,
}

type metricsUpdater interface {
	UpdateMetrics(
		context.Context,
		*metricspb.UpdateMetricsRequest,
		...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error)
}

// GRPCSender отправляет batch метрик на Сервер через gRPC.
type GRPCSender struct {
	client      metricsUpdater
	closer      io.Closer
	localIP     func() (string, error)
	retryDelays []time.Duration
}

var _ service.MetricsSender = (*GRPCSender)(nil)

// NewGRPCSender создаёт gRPC-отправитель метрик и одно переиспользуемое соединение.
func NewGRPCSender(serverAddr string) (*GRPCSender, error) {
	if serverAddr == "" {
		return nil, fmt.Errorf("server address must not be empty")
	}

	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("create gRPC client: %w", err)
	}

	return newGRPCSender(
		metricspb.NewMetricsClient(conn),
		conn,
		agenttransport.LocalIP,
	), nil
}

func newGRPCSender(
	client metricsUpdater,
	closer io.Closer,
	localIP func() (string, error),
) *GRPCSender {
	return &GRPCSender{
		client:      client,
		closer:      closer,
		localIP:     localIP,
		retryDelays: append([]time.Duration(nil), defaultRetryDelays...),
	}
}

// Send отправляет все метрики одним вызовом Metrics.UpdateMetrics.
func (s *GRPCSender) Send(ctx context.Context, metrics []model.Metrics) error {
	rq, err := requestFromMetrics(metrics)
	if err != nil {
		return err
	}

	realIP, err := s.localIP()
	if err != nil {
		return fmt.Errorf("get local IP: %w", err)
	}

	ctx = contextWithRealIP(ctx, realIP)

	var lastErr error

	for attempt := 0; ; attempt++ {
		_, lastErr = s.client.UpdateMetrics(ctx, rq)
		if lastErr == nil {
			return nil
		}

		if attempt >= len(s.retryDelays) || !isRetriableGRPCError(lastErr) {
			return fmt.Errorf("send metrics batch: %w", lastErr)
		}

		if err := waitRetry(ctx, s.retryDelays[attempt]); err != nil {
			return fmt.Errorf("send metrics batch: %w", err)
		}
	}
}

// Close закрывает gRPC-соединение Агента.
func (s *GRPCSender) Close() error {
	if s.closer == nil {
		return nil
	}

	return s.closer.Close()
}

func contextWithRealIP(ctx context.Context, realIP string) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = md.Copy()
	} else {
		md = metadata.MD{}
	}
	md.Set(protocol.MetadataRealIP, realIP)

	return metadata.NewOutgoingContext(ctx, md)
}

func isRetriableGRPCError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	switch status.Code(err) {
	case codes.Unavailable, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
