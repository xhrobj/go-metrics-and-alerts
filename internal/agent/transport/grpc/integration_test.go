package grpctransport

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type recordingMetricsServer struct {
	metricspb.UnimplementedMetricsServer

	rqCh chan *metricspb.UpdateMetricsRequest
	ipCh chan string
}

func (s *recordingMetricsServer) UpdateMetrics(
	ctx context.Context,
	rq *metricspb.UpdateMetricsRequest,
) (*metricspb.UpdateMetricsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	s.rqCh <- rq
	s.ipCh <- firstMetadataValue(md, protocol.MetadataRealIP)

	return metricspb.UpdateMetricsResponse_builder{}.Build(), nil
}

func TestGRPCSenderSendsBatchThroughRealConnection(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	recorder := &recordingMetricsServer{
		rqCh: make(chan *metricspb.UpdateMetricsRequest, 1),
		ipCh: make(chan string, 1),
	}
	metricspb.RegisterMetricsServer(srv, recorder)

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- srv.Serve(listener)
	}()

	t.Cleanup(func() {
		srv.Stop()
		_ = listener.Close()
		require.NoError(t, <-serveErrCh)
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	require.NoError(t, err)

	sender := newGRPCSender(
		metricspb.NewMetricsClient(conn),
		conn,
		func() (string, error) {
			return "192.168.1.42", nil
		},
	)
	t.Cleanup(func() {
		require.NoError(t, sender.Close())
	})

	gaugeValue := 5.11
	counterDelta := int64(42)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = sender.Send(ctx, []model.Metrics{
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

	select {
	case rq := <-recorder.rqCh:
		require.Len(t, rq.GetMetrics(), 2)
		require.Equal(t, gaugeValue, rq.GetMetrics()[0].GetValue())
		require.Equal(t, counterDelta, rq.GetMetrics()[1].GetDelta())
	case <-ctx.Done():
		t.Fatal("timeout waiting for gRPC request")
	}

	select {
	case ip := <-recorder.ipCh:
		require.Equal(t, "192.168.1.42", ip)
	case <-ctx.Done():
		t.Fatal("timeout waiting for gRPC metadata")
	}
}

func firstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}

	return values[0]
}
