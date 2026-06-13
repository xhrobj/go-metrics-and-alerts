package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
)

// TestWithDecryptionDecryptsBody проверяет передачу расшифрованного body
// следующему HTTP-handler'у.
func TestWithDecryptionDecryptsBody(t *testing.T) {
	publicKey, err := encryption.LoadPublicKey(filepath.Join(
		"..",
		"encryption",
		"testdata",
		"public.pem",
	))
	if err != nil {
		t.Fatalf("LoadPublicKey() error = %v", err)
	}

	privateKey, err := encryption.LoadPrivateKey(filepath.Join(
		"..",
		"encryption",
		"testdata",
		"private.pem",
	))
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	want := []byte(`{"id":"Alloc","type":"gauge","value":5.11}`)

	encryptedBody, err := encryption.Encrypt(want, publicKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	var got []byte

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		if value := r.Header.Get(contentEncryptionHeader); value != "" {
			t.Fatalf("%s = %q, want empty", contentEncryptionHeader, value)
		}

		w.WriteHeader(http.StatusOK)
	})

	rq := httptest.NewRequest(http.MethodPost, "/updates", bytes.NewReader(encryptedBody))
	rq.Header.Set(contentEncryptionHeader, hybridEncryption)

	rs := httptest.NewRecorder()

	WithDecryption(privateKey)(next).ServeHTTP(rs, rq)

	if gotStatus, wantStatus := rs.Code, http.StatusOK; gotStatus != wantStatus {
		t.Fatalf("status = %d, want %d", gotStatus, wantStatus)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("body = %s, want %s", got, want)
	}
}

// TestWithDecryptionPassesPlaintextBody проверяет, что запрос без заголовка
// Content-Encryption обрабатывается как раньше.
func TestWithDecryptionPassesPlaintextBody(t *testing.T) {
	want := []byte(`{"id":"PollCount","type":"counter","delta":42}`)

	var got []byte

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		got, err = io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

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

// TestWithDecryptionRejectsWrongPrivateKey проверяет, что тело,
// зашифрованное для другого RSA-ключа, отклоняется.
func TestWithDecryptionRejectsWrongPrivateKey(t *testing.T) {
	publicKey, err := encryption.LoadPublicKey(filepath.Join(
		"..",
		"encryption",
		"testdata",
		"public.pem",
	))
	if err != nil {
		t.Fatalf("LoadPublicKey() error = %v", err)
	}

	encryptedBody, err := encryption.Encrypt([]byte("encrypted data"), publicKey)
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
	rq.Header.Set(contentEncryptionHeader, hybridEncryption)

	rs := httptest.NewRecorder()

	WithDecryption(wrongPrivateKey)(next).ServeHTTP(rs, rq)

	if got, want := rs.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	if handlerCalled {
		t.Fatal("next handler called, want false")
	}
}
