package server

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metricspb "github.com/xhrobj/go-metrics-and-alerts/internal/proto"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestNewReturnsErrorForInvalidTrustedSubnet(t *testing.T) {
	_, err := New(config.ServerConfig{
		TrustedSubnet: "invalid-cidr",
	}, zap.NewNop())

	require.ErrorContains(t, err, "parse trusted subnet")
}

func TestNewReturnsErrorWhenGRPCAddressIsAlreadyInUse(t *testing.T) {
	occupiedListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = occupiedListener.Close()
	})

	_, err = New(config.ServerConfig{
		HTTPAddr: "127.0.0.1:0",
		GRPCAddr: occupiedListener.Addr().String(),
	}, zap.NewNop())

	require.ErrorContains(t, err, "listen gRPC")
}

func TestServerRunsHTTPAndGRPC(t *testing.T) {
	app, err := New(config.ServerConfig{
		HTTPAddr:           "127.0.0.1:0",
		GRPCAddr:           "127.0.0.1:0",
		StoreIntervalInSec: 300,
	}, zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(app.Close)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- app.Run(ctx)
	}()

	conn, err := grpc.NewClient(
		app.grpcListener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	rpcCtx, rpcCancel := context.WithTimeout(context.Background(), time.Second)
	defer rpcCancel()

	_, err = metricspb.NewMetricsClient(conn).UpdateMetrics(
		rpcCtx,
		metricspb.UpdateMetricsRequest_builder{
			Metrics: []*metricspb.Metric{
				metricspb.Metric_builder{
					Id:    "sharedCounter",
					Type:  metricspb.Metric_COUNTER,
					Delta: 42,
				}.Build(),
			},
		}.Build(),
	)
	require.NoError(t, err)

	httpClient := &http.Client{Timeout: time.Second}

	rq, err := http.NewRequest(
		http.MethodPost,
		"http://"+app.httpListener.Addr().String()+"/update/counter/sharedCounter/42",
		nil,
	)
	require.NoError(t, err)

	rs, err := httpClient.Do(rq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rs.StatusCode)
	require.NoError(t, rs.Body.Close())

	rs, err = httpClient.Get(
		"http://" + app.httpListener.Addr().String() + "/value/counter/sharedCounter",
	)
	require.NoError(t, err)

	body, err := io.ReadAll(rs.Body)
	require.NoError(t, err)
	require.NoError(t, rs.Body.Close())
	require.Equal(t, http.StatusOK, rs.StatusCode)
	require.Equal(t, "84", string(body))

	cancel()

	select {
	case err := <-runErrCh:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Server did not stop")
	}
}

func TestServeHTTPWaitsForActiveRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	allowRequestToFinish := make(chan struct{})

	h := http.HandlerFunc(func(rs http.ResponseWriter, _ *http.Request) {
		close(requestStarted)
		<-allowRequestToFinish
		rs.WriteHeader(http.StatusOK)
	})

	httpListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = httpListener.Close()
	})

	srv := &http.Server{Handler: h}

	ctx, cancel := context.WithCancel(context.Background())
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- serveHTTP(ctx, srv, httpListener, zap.NewNop())
	}()

	requestErrCh := make(chan error, 1)
	go func() {
		rs, err := http.Get("http://" + httpListener.Addr().String())
		if err != nil {
			requestErrCh <- err
			return
		}
		_, copyErr := io.Copy(io.Discard, rs.Body)
		closeErr := rs.Body.Close()

		requestErrCh <- errors.Join(copyErr, closeErr)
	}()

	select {
	case <-requestStarted:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}

	cancel()

	select {
	case err := <-serveErrCh:
		t.Fatalf("serveHTTP returned before active request finished: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(allowRequestToFinish)

	require.NoError(t, <-requestErrCh)
	require.NoError(t, <-serveErrCh)
}
