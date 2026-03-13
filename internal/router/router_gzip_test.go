package router_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
	"github.com/xhrobj/go-metrics-and-alerts/internal/service"
)

func TestGzipMiddleware(t *testing.T) {
	repo := repository.NewMemStorage()
	srv := service.NewMetricsService(repo)
	h := handler.New(srv)
	log := zap.NewNop()

	r := router.New(h, log)

	ts := httptest.NewServer(r)
	defer ts.Close()

	t.Run("server accepts gzip request body", func(t *testing.T) {
		requestBody := `{"id":"test","type":"gauge","value":1}`

		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)

		_, err := zw.Write([]byte(requestBody))
		require.NoError(t, err)

		err = zw.Close()
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, ts.URL+"/update", &buf)
		require.NoError(t, err)

		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		require.JSONEq(t, `{"id":"test","type":"gauge","value":1}`, string(body))
	})

	t.Run("server returns gzip response when client supports gzip", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
		require.NoError(t, err)

		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

		zr, err := gzip.NewReader(resp.Body)
		require.NoError(t, err)
		defer zr.Close()

		body, err := io.ReadAll(zr)
		require.NoError(t, err)

		require.NotEmpty(t, body)
	})

	t.Run("server does not gzip non json/html responses", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
		require.NoError(t, err)

		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		// gzip не должен включаться
		require.NotEqual(t, "gzip", resp.Header.Get("Content-Encoding"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		require.Equal(t, "pong", string(body))
	})
}
