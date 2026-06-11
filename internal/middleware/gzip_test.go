package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithGzipCompressesJSONResponse(t *testing.T) {
	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}))

	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rq.Header.Set("Accept-Encoding", "gzip")
	rs := httptest.NewRecorder()

	handler.ServeHTTP(rs, rq)

	if got, want := rs.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if got, want := rs.Header().Get("Content-Encoding"), "gzip"; got != want {
		t.Fatalf("Content-Encoding = %q, want %q", got, want)
	}

	zr, err := gzip.NewReader(rs.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer func() {
		if err := zr.Close(); err != nil {
			t.Fatalf("gzip reader Close() error = %v", err)
		}
	}()

	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if got, want := string(body), `{"status":"ok"}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWithGzipDoesNotCompressPlainTextResponse(t *testing.T) {
	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		if _, err := w.Write([]byte("plain response")); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}))

	rq := httptest.NewRequest(http.MethodGet, "/", nil)
	rq.Header.Set("Accept-Encoding", "gzip")
	rs := httptest.NewRecorder()

	handler.ServeHTTP(rs, rq)

	if got, want := rs.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if got := rs.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}

	if got, want := rs.Body.String(), "plain response"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWithGzipDecompressesRequestBody(t *testing.T) {
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(`{"id":"PollCount"}`)); err != nil {
		t.Fatalf("gzip Write() error = %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip Close() error = %v", err)
	}

	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		body, err := io.ReadAll(rq.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if got, want := string(body), `{"id":"PollCount"}`; got != want {
			t.Fatalf("request body = %q, want %q", got, want)
		}

		w.WriteHeader(http.StatusAccepted)
	}))

	rq := httptest.NewRequest(http.MethodPost, "/", &buf)
	rq.Header.Set("Content-Encoding", "gzip")
	rs := httptest.NewRecorder()

	handler.ServeHTTP(rs, rq)

	if got, want := rs.Code, http.StatusAccepted; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}

func TestWithGzipRejectsInvalidGzipRequestBody(t *testing.T) {
	called := false

	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		called = true
	}))

	rq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not gzip"))
	rq.Header.Set("Content-Encoding", "gzip")
	rs := httptest.NewRecorder()

	handler.ServeHTTP(rs, rq)

	if called {
		t.Fatal("handler was called for invalid gzip body")
	}

	if got, want := rs.Code, http.StatusInternalServerError; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
}
