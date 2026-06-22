package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption/testkeys"
)

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (failingReadCloser) Close() error {
	return nil
}

// TestWithDecryptionDecryptsBody проверяет передачу расшифрованного body
// следующему HTTP-handler'у.
func TestWithDecryptionDecryptsBody(t *testing.T) {
	pair := testkeys.Generate(t)

	want := []byte(`{"id":"Alloc","type":"gauge","value":5.11}`)

	encryptedBody, err := encryption.Encrypt(want, pair.PublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	originalBody := &trackingReadCloser{
		Reader: bytes.NewReader(encryptedBody),
	}

	var got []byte

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		got = body

		if value := r.Header.Get(encryption.HeaderContentEncryption); value != "" {
			t.Fatalf(
				"%s = %q, want empty",
				encryption.HeaderContentEncryption,
				value,
			)
		}

		if got, want := r.Header.Get("Content-Encoding"), "gzip"; got != want {
			t.Fatalf("Content-Encoding = %q, want %q", got, want)
		}

		if got, want := r.ContentLength, int64(len(body)); got != want {
			t.Fatalf("ContentLength = %d, want %d", got, want)
		}

		w.WriteHeader(http.StatusOK)
	})

	rq := httptest.NewRequest(http.MethodPost, "/updates", nil)
	rq.Body = originalBody
	rq.ContentLength = int64(len(encryptedBody))
	rq.Header.Set(
		encryption.HeaderContentEncryption,
		encryption.SchemeRSAOAEPWithAESGCM,
	)
	rq.Header.Set("Content-Encoding", "gzip")

	rs := httptest.NewRecorder()

	WithDecryption(pair.PrivateKey)(next).ServeHTTP(rs, rq)

	if got, want := rs.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("body = %s, want %s", got, want)
	}

	if !originalBody.closed {
		t.Fatal("original body closed = false, want true")
	}
}

// TestWithDecryptionPassesPlaintextBody проверяет, что запрос без заголовка
// Content-Encryption обрабатывается как раньше.
func TestWithDecryptionPassesPlaintextBody(t *testing.T) {
	want := []byte(`{"id":"PollCount","type":"counter","delta":42}`)

	var got []byte

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		got = body
		w.WriteHeader(http.StatusOK)
	})

	rq := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(want))
	rs := httptest.NewRecorder()

	WithDecryption(nil)(next).ServeHTTP(rs, rq)

	if gotStatus, wantStatus := rs.Code, http.StatusOK; gotStatus != wantStatus {
		t.Fatalf("status = %d, want %d", gotStatus, wantStatus)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

// TestWithDecryptionRejectsInvalidRequests проверяет отклонение
// некорректных зашифрованных запросов до вызова следующего handler'а.
func TestWithDecryptionRejectsInvalidRequests(t *testing.T) {
	pair := testkeys.Generate(t)

	encryptedBody, err := encryption.Encrypt(
		[]byte("encrypted data"),
		pair.PublicKey,
	)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	tests := []struct {
		name       string
		privateKey *rsa.PrivateKey
		scheme     string
		body       func() io.ReadCloser
	}{
		{
			name:       "missing private key",
			privateKey: nil,
			scheme:     encryption.SchemeRSAOAEPWithAESGCM,
			body: func() io.ReadCloser {
				return io.NopCloser(bytes.NewReader(encryptedBody))
			},
		},
		{
			name:       "unsupported encryption scheme",
			privateKey: pair.PrivateKey,
			scheme:     "unsupported",
			body: func() io.ReadCloser {
				return io.NopCloser(bytes.NewReader(encryptedBody))
			},
		},
		{
			name:       "missing body",
			privateKey: pair.PrivateKey,
			scheme:     encryption.SchemeRSAOAEPWithAESGCM,
			body: func() io.ReadCloser {
				return nil
			},
		},
		{
			name:       "damaged envelope",
			privateKey: pair.PrivateKey,
			scheme:     encryption.SchemeRSAOAEPWithAESGCM,
			body: func() io.ReadCloser {
				return io.NopCloser(bytes.NewReader([]byte("damaged envelope")))
			},
		},
		{
			name:       "body read error",
			privateKey: pair.PrivateKey,
			scheme:     encryption.SchemeRSAOAEPWithAESGCM,
			body: func() io.ReadCloser {
				return failingReadCloser{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				w.WriteHeader(http.StatusOK)
			})

			rq := httptest.NewRequest(http.MethodPost, "/updates", nil)
			rq.Body = tt.body()
			rq.Header.Set(encryption.HeaderContentEncryption, tt.scheme)

			rs := httptest.NewRecorder()

			WithDecryption(tt.privateKey)(next).ServeHTTP(rs, rq)

			if got, want := rs.Code, http.StatusBadRequest; got != want {
				t.Fatalf("status = %d, want %d", got, want)
			}

			if handlerCalled {
				t.Fatal("next handler called = true, want false")
			}
		})
	}
}

// TestWithDecryptionRejectsWrongPrivateKey проверяет, что тело,
// зашифрованное для другого RSA-ключа, отклоняется.
func TestWithDecryptionRejectsWrongPrivateKey(t *testing.T) {
	pair := testkeys.Generate(t)

	encryptedBody, err := encryption.Encrypt(
		[]byte("encrypted data"),
		pair.PublicKey,
	)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	wrongPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	handlerCalled := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	rq := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(encryptedBody))
	rq.Header.Set(
		encryption.HeaderContentEncryption,
		encryption.SchemeRSAOAEPWithAESGCM,
	)

	rs := httptest.NewRecorder()

	WithDecryption(wrongPrivateKey)(next).ServeHTTP(rs, rq)

	if got, want := rs.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if handlerCalled {
		t.Fatal("next handler called = true, want false")
	}
}
