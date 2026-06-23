package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufconnSize = 1024 * 1024

type blockingMetricsServer struct {
	metricspb.UnimplementedMetricsServer

	requestStarted       chan struct{}
	allowRequestToFinish chan struct{}
}

func (s *blockingMetricsServer) UpdateMetrics(
	ctx context.Context,
	_ *metricspb.UpdateMetricsRequest,
) (*metricspb.UpdateMetricsResponse, error) {
	close(s.requestStarted)

	select {
	case <-s.allowRequestToFinish:
		return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestServeGRPCWaitsForActiveRequest(t *testing.T) {
	listener := bufconn.Listen(bufconnSize)
	t.Cleanup(func() {
		_ = listener.Close()
	})

	service := &blockingMetricsServer{
		requestStarted:       make(chan struct{}),
		allowRequestToFinish: make(chan struct{}),
	}

	srv := grpc.NewServer()
	metricspb.RegisterMetricsServer(srv, service)

	ctx, cancel := context.WithCancel(context.Background())
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- serveGRPC(ctx, srv, listener, zap.NewNop())
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	rpcErrCh := make(chan error, 1)
	go func() {
		_, err := metricspb.NewMetricsClient(conn).UpdateMetrics(
			context.Background(),
			metricspb.UpdateMetricsRequest_builder{}.Build(),
		)
		rpcErrCh <- err
	}()

	select {
	case <-service.requestStarted:
	case <-time.After(time.Second):
		t.Fatal("gRPC request did not start")
	}

	cancel()

	select {
	case err := <-serveErrCh:
		t.Fatalf("serveGRPC returned before active request finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(service.allowRequestToFinish)

	require.NoError(t, <-rpcErrCh)
	require.NoError(t, <-serveErrCh)
}

func TestStopGRPCForcesStopAfterTimeout(t *testing.T) {
	listener := bufconn.Listen(bufconnSize)
	t.Cleanup(func() {
		_ = listener.Close()
	})

	service := &blockingMetricsServer{
		requestStarted:       make(chan struct{}),
		allowRequestToFinish: make(chan struct{}),
	}

	srv := grpc.NewServer()
	metricspb.RegisterMetricsServer(srv, service)

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	rpcErrCh := make(chan error, 1)
	go func() {
		_, err := metricspb.NewMetricsClient(conn).UpdateMetrics(
			context.Background(),
			metricspb.UpdateMetricsRequest_builder{}.Build(),
		)
		rpcErrCh <- err
	}()

	select {
	case <-service.requestStarted:
	case <-time.After(time.Second):
		t.Fatal("gRPC request did not start")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = stopGRPC(shutdownCtx, srv)
	require.ErrorContains(t, err, "shutdown gRPC server: context deadline exceeded")
	require.Error(t, <-rpcErrCh)

	require.NoError(t, <-serveErrCh)
}
