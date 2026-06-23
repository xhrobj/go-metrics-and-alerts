package grpctransport

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type metricsClientFunc func(
	context.Context,
	*metricspb.UpdateMetricsRequest,
	...grpc.CallOption,
) (*metricspb.UpdateMetricsResponse, error)

func (f metricsClientFunc) UpdateMetrics(
	ctx context.Context,
	rq *metricspb.UpdateMetricsRequest,
	opts ...grpc.CallOption,
) (*metricspb.UpdateMetricsResponse, error) {
	return f(ctx, rq, opts...)
}

type closerStub struct {
	calls atomic.Int32
	err   error
}

func (c *closerStub) Close() error {
	c.calls.Add(1)
	return c.err
}

func TestGRPCSenderSend(t *testing.T) {
	gaugeValue := 5.11
	counterDelta := int64(42)

	client := metricsClientFunc(func(
		ctx context.Context,
		rq *metricspb.UpdateMetricsRequest,
		_ ...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		require.Equal(t, []string{"192.168.1.42"}, md.Get(protocol.MetadataRealIP))

		metrics := rq.GetMetrics()
		require.Len(t, metrics, 2)
		require.Equal(t, "Alloc", metrics[0].GetId())
		require.Equal(t, metricspb.Metric_GAUGE, metrics[0].GetType())
		require.Equal(t, gaugeValue, metrics[0].GetValue())
		require.Equal(t, "PollCount", metrics[1].GetId())
		require.Equal(t, metricspb.Metric_COUNTER, metrics[1].GetType())
		require.Equal(t, counterDelta, metrics[1].GetDelta())

		return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "192.168.1.42", nil
	})

	err := sender.Send(context.Background(), []model.Metrics{
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
}

func TestGRPCSenderPreservesOutgoingMetadata(t *testing.T) {
	value := 5.11

	client := metricsClientFunc(func(
		ctx context.Context,
		_ *metricspb.UpdateMetricsRequest,
		_ ...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		md, ok := metadata.FromOutgoingContext(ctx)
		require.True(t, ok)
		require.Equal(t, []string{"value"}, md.Get("existing"))
		require.Equal(t, []string{"192.168.1.42"}, md.Get(protocol.MetadataRealIP))

		return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "192.168.1.42", nil
	})

	ctx := metadata.NewOutgoingContext(
		context.Background(),
		metadata.Pairs("existing", "value", protocol.MetadataRealIP, "old"),
	)

	err := sender.Send(ctx, []model.Metrics{{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: &value,
	}})
	require.NoError(t, err)
}

func TestGRPCSenderRetriesTemporaryErrors(t *testing.T) {
	value := 5.11
	var calls atomic.Int32

	client := metricsClientFunc(func(
		context.Context,
		*metricspb.UpdateMetricsRequest,
		...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		switch calls.Add(1) {
		case 1:
			return nil, status.Error(codes.Unavailable, "temporarily unavailable")
		case 2:
			return nil, status.Error(codes.ResourceExhausted, "try later")
		default:
			return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
		}
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "192.168.1.42", nil
	})
	sender.retryDelays = []time.Duration{0, 0}

	err := sender.Send(context.Background(), []model.Metrics{{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: &value,
	}})
	require.NoError(t, err)
	require.Equal(t, int32(3), calls.Load())
}

func TestGRPCSenderDoesNotRetryPermanentError(t *testing.T) {
	value := 5.11
	var calls atomic.Int32

	client := metricsClientFunc(func(
		context.Context,
		*metricspb.UpdateMetricsRequest,
		...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		calls.Add(1)
		return nil, status.Error(codes.PermissionDenied, "access denied")
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "192.168.1.42", nil
	})
	sender.retryDelays = []time.Duration{0, 0, 0}

	err := sender.Send(context.Background(), []model.Metrics{{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: &value,
	}})
	require.Error(t, err)
	require.Equal(t, codes.PermissionDenied, status.Code(errors.Unwrap(err)))
	require.Equal(t, int32(1), calls.Load())
}

func TestGRPCSenderStopsRetryWhenContextIsCanceled(t *testing.T) {
	value := 5.11
	firstAttempt := make(chan struct{})
	var calls atomic.Int32

	client := metricsClientFunc(func(
		context.Context,
		*metricspb.UpdateMetricsRequest,
		...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		if calls.Add(1) == 1 {
			close(firstAttempt)
		}
		return nil, status.Error(codes.Unavailable, "temporarily unavailable")
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "192.168.1.42", nil
	})
	sender.retryDelays = []time.Duration{time.Hour}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)

	go func() {
		errCh <- sender.Send(ctx, []model.Metrics{{
			ID:    "Alloc",
			MType: model.Gauge,
			Value: &value,
		}})
	}()

	<-firstAttempt
	cancel()

	err := <-errCh
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(1), calls.Load())
}

func TestGRPCSenderReturnsLocalIPErrorBeforeRPC(t *testing.T) {
	value := 5.11
	var calls atomic.Int32

	client := metricsClientFunc(func(
		context.Context,
		*metricspb.UpdateMetricsRequest,
		...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		calls.Add(1)
		return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "", errors.New("network unavailable")
	})

	err := sender.Send(context.Background(), []model.Metrics{{
		ID:    "Alloc",
		MType: model.Gauge,
		Value: &value,
	}})
	require.ErrorContains(t, err, "get local IP")
	require.Zero(t, calls.Load())
}

func TestGRPCSenderSupportsConcurrentSend(t *testing.T) {
	value := 5.11
	var calls atomic.Int32

	client := metricsClientFunc(func(
		context.Context,
		*metricspb.UpdateMetricsRequest,
		...grpc.CallOption,
	) (*metricspb.UpdateMetricsResponse, error) {
		calls.Add(1)
		return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
	})

	sender := newGRPCSender(client, nil, func() (string, error) {
		return "192.168.1.42", nil
	})

	const goroutines = 42

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- sender.Send(context.Background(), []model.Metrics{{
				ID:    "Alloc",
				MType: model.Gauge,
				Value: &value,
			}})
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}

	require.Equal(t, int32(goroutines), calls.Load())
}

func TestGRPCSenderClose(t *testing.T) {
	wantErr := errors.New("close failed")
	closer := &closerStub{err: wantErr}
	sender := newGRPCSender(nil, closer, nil)

	err := sender.Close()
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, int32(1), closer.calls.Load())
}

func TestNewGRPCSenderRejectsEmptyAddress(t *testing.T) {
	_, err := NewGRPCSender("")
	require.ErrorContains(t, err, "server address")
}

func TestGRPCSenderCloseWithoutCloser(t *testing.T) {
	sender := newGRPCSender(nil, nil, nil)

	require.NoError(t, sender.Close())
}
