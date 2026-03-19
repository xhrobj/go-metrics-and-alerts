package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	internalhash "github.com/xhrobj/go-metrics-and-alerts/internal/hash"
)

func TestWithHash_ValidRequest(t *testing.T) {
	hashKey := "secret-key"
	body := []byte(`{"id":"test","type":"gauge","value":123}`)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("HashSHA256", internalhash.CalcHash(body, hashKey))

	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := WithHash(hashKey)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestWithHash_InvalidRequest(t *testing.T) {
	hashKey := "secret-key"
	body := []byte(`{"id":"test","type":"gauge","value":123}`)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	req.Header.Set("HashSHA256", "wrong-hash")

	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler must not be called for invalid hash")
	})

	handler := WithHash(hashKey)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestWithHash_RestoresRequestBody(t *testing.T) {
	hashKey := "secret-key"
	body := []byte(`{"id":"test","type":"gauge","value":123}`)

	req := httptest.NewRequest(http.MethodPost, "/update", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body in next handler: %v", err)
		}

		if !bytes.Equal(gotBody, body) {
			t.Fatalf("expected body %q, got %q", body, gotBody)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := WithHash(hashKey)(next)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestWithHash_SetsResponseHashHeader(t *testing.T) {
	hashKey := "god"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	responseBody := []byte("hello")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBody)
	})

	handler := WithHash(hashKey)(next)
	handler.ServeHTTP(rec, req)

	gotHash := rec.Header().Get("HashSHA256")
	wantHash := internalhash.CalcHash(responseBody, hashKey)

	if gotHash != wantHash {
		t.Fatalf("expected response hash %q, got %q", wantHash, gotHash)
	}
}
