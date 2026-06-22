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
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"go.uber.org/zap"
)

func TestNewReturnsErrorForInvalidTrustedSubnet(t *testing.T) {
	_, err := New(config.ServerConfig{
		TrustedSubnet: "invalid-cidr",
	}, zap.NewNop())

	require.ErrorContains(t, err, "parse trusted subnet")
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
