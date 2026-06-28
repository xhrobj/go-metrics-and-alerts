package httptransport

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption/testkeys"
	metricshash "github.com/xhrobj/go-metrics-and-alerts/internal/hash"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
)

func TestNewHTTPSender(t *testing.T) {
	t.Run("adds HTTP scheme", func(t *testing.T) {
		sender, err := NewHTTPSender("localhost:8080", "", "")
		require.NoError(t, err)
		require.Equal(t, "http://localhost:8080", sender.baseURL)
		require.Nil(t, sender.publicKey)
	})

	t.Run("preserves existing scheme", func(t *testing.T) {
		sender, err := NewHTTPSender("https://example.com", "", "")
		require.NoError(t, err)
		require.Equal(t, "https://example.com", sender.baseURL)
	})

	t.Run("loads public key", func(t *testing.T) {
		keys := testkeys.Generate(t)

		sender, err := NewHTTPSender("localhost:8080", "", keys.PublicKeyPath)
		require.NoError(t, err)
		require.NotNil(t, sender.publicKey)
	})

	t.Run("returns public key error", func(t *testing.T) {
		_, err := NewHTTPSender("localhost:8080", "", "missing-public-key.pem")
		require.ErrorContains(t, err, "load public key")
	})
}

func TestHTTPSenderSend(t *testing.T) {
	metrics := testMetrics()
	hashKey := "secret"

	requestCh := make(chan capturedRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(rs http.ResponseWriter, rq *http.Request) {
		body, err := io.ReadAll(rq.Body)
		if err != nil {
			rs.WriteHeader(http.StatusInternalServerError)
			return
		}

		requestCh <- capturedRequest{
			method:  rq.Method,
			path:    rq.URL.Path,
			headers: rq.Header.Clone(),
			body:    body,
		}
		rs.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	sender, err := NewHTTPSender(srv.URL, hashKey, "")
	require.NoError(t, err)

	require.NoError(t, sender.Send(context.Background(), metrics))

	captured := <-requestCh
	require.Equal(t, http.MethodPost, captured.method)
	require.Equal(t, "/updates", captured.path)
	require.Equal(t, protocol.ContentTypeJSON, captured.headers.Get(protocol.HeaderContentType))
	require.Equal(t, protocol.EncodingGzip, captured.headers.Get(protocol.HeaderContentEncoding))
	require.Empty(t, captured.headers.Get(protocol.HeaderContentEncryption))
	require.Equal(
		t,
		metricshash.CalcHash(captured.body, hashKey),
		captured.headers.Get(protocol.HeaderHashSHA256),
	)
	require.NotNil(t, net.ParseIP(captured.headers.Get(protocol.HeaderRealIP)))
	require.Equal(t, metrics, decodeMetrics(t, captured.body))
}

func TestHTTPSenderSendEncrypted(t *testing.T) {
	keys := testkeys.Generate(t)
	metrics := testMetrics()
	hashKey := "secret"

	requestCh := make(chan capturedRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(rs http.ResponseWriter, rq *http.Request) {
		body, err := io.ReadAll(rq.Body)
		if err != nil {
			rs.WriteHeader(http.StatusInternalServerError)
			return
		}

		requestCh <- capturedRequest{
			headers: rq.Header.Clone(),
			body:    body,
		}
		rs.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	sender, err := NewHTTPSender(srv.URL, hashKey, keys.PublicKeyPath)
	require.NoError(t, err)

	require.NoError(t, sender.Send(context.Background(), metrics))

	captured := <-requestCh
	require.Equal(
		t,
		encryption.SchemeRSAOAEPWithAESGCM,
		captured.headers.Get(protocol.HeaderContentEncryption),
	)
	require.Equal(
		t,
		metricshash.CalcHash(captured.body, hashKey),
		captured.headers.Get(protocol.HeaderHashSHA256),
	)

	decryptedBody, err := encryption.Decrypt(captured.body, keys.PrivateKey)
	require.NoError(t, err)
	require.Equal(t, metrics, decodeMetrics(t, decryptedBody))
}

func TestHTTPSenderSendDoesNotRetryBadRequest(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(rs http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		rs.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)

	sender, err := NewHTTPSender(srv.URL, "", "")
	require.NoError(t, err)

	err = sender.Send(context.Background(), testMetrics())
	require.ErrorContains(t, err, "unexpected status code: 400")
	require.Equal(t, int32(1), attempts.Load())
}

func TestHTTPSenderSendRetriesServerError(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(rs http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			rs.WriteHeader(http.StatusInternalServerError)
			return
		}

		rs.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	sender, err := NewHTTPSender(srv.URL, "", "")
	require.NoError(t, err)

	require.NoError(t, sender.Send(context.Background(), testMetrics()))
	require.Equal(t, int32(2), attempts.Load())
}

func TestHTTPSenderSendStopsRetryWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(rs http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		cancel()
		rs.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	sender, err := NewHTTPSender(srv.URL, "", "")
	require.NoError(t, err)

	err = sender.Send(ctx, testMetrics())
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(1), attempts.Load())
}

func TestIsRetriableSendError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "too many requests",
			err:  &unexpectedStatusError{statusCode: http.StatusTooManyRequests},
			want: true,
		},
		{
			name: "server error",
			err:  &unexpectedStatusError{statusCode: http.StatusServiceUnavailable},
			want: true,
		},
		{
			name: "client error",
			err:  &unexpectedStatusError{statusCode: http.StatusBadRequest},
			want: false,
		},
		{
			name: "network error",
			err:  &net.DNSError{IsTimeout: true},
			want: true,
		},
		{
			name: "ordinary error",
			err:  errors.New("boom"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isRetriableSendError(tt.err))
		})
	}
}

func TestWaitRetry(t *testing.T) {
	t.Run("delay elapsed", func(t *testing.T) {
		require.NoError(t, waitRetry(context.Background(), time.Millisecond))
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		require.ErrorIs(t, waitRetry(ctx, time.Second), context.Canceled)
	})
}

type capturedRequest struct {
	method  string
	path    string
	headers http.Header
	body    []byte
}

func testMetrics() []model.Metrics {
	gaugeValue := 5.11
	counterDelta := int64(42)

	return []model.Metrics{
		{
			ID:    "gaugeMetric",
			MType: model.Gauge,
			Value: &gaugeValue,
		},
		{
			ID:    "counterMetric",
			MType: model.Counter,
			Delta: &counterDelta,
		},
	}
}

func decodeMetrics(t *testing.T, body []byte) []model.Metrics {
	t.Helper()

	zr, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)

	decompressedBody, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.NoError(t, zr.Close())

	var metrics []model.Metrics
	require.NoError(t, json.Unmarshal(decompressedBody, &metrics))

	return metrics
}
