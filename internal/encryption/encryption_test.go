package encryption

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption/testkeys"
)

func TestLoadPublicKey(t *testing.T) {
	pair := testkeys.Generate(t)

	publicKey, err := LoadPublicKey(pair.PublicKeyPath)
	if err != nil {
		t.Fatalf("LoadPublicKey() error = %v", err)
	}

	if publicKey == nil {
		t.Fatal("LoadPublicKey() = nil")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	pair := testkeys.Generate(t)

	privateKey, err := LoadPrivateKey(pair.PrivateKeyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	if privateKey == nil {
		t.Fatal("LoadPrivateKey() = nil")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	pair := testkeys.Generate(t)

	publicKey, err := LoadPublicKey(pair.PublicKeyPath)
	if err != nil {
		t.Fatalf("LoadPublicKey() error = %v", err)
	}

	privateKey, err := LoadPrivateKey(pair.PrivateKeyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	want := []byte(`{"id":"Alloc","type":"gauge","value":5.11}`)

	encryptedData, err := Encrypt(want, publicKey)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	got, err := Decrypt(encryptedData, privateKey)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("Decrypt() = %s, want %s", got, want)
	}
}

func TestDecryptInvalidPayload(t *testing.T) {
	pair := testkeys.Generate(t)

	privateKey, err := LoadPrivateKey(pair.PrivateKeyPath)
	if err != nil {
		t.Fatalf("LoadPrivateKey() error = %v", err)
	}

	_, err = Decrypt([]byte("invalid payload"), privateKey)
	if err == nil {
		t.Fatal("Decrypt() error = nil, want error")
	}
}

func TestLoadPublicKeyInvalidPEM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "public.pem")

	if err := os.WriteFile(path, []byte("invalid pem"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadPublicKey(path)
	if err == nil {
		t.Fatal("LoadPublicKey() error = nil, want error")
	}
}

func TestLoadPrivateKeyInvalidPEM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "private.pem")

	if err := os.WriteFile(path, []byte("invalid pem"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadPrivateKey(path)
	if err == nil {
		t.Fatal("LoadPrivateKey() error = nil, want error")
	}
}

func TestDecryptInvalidNonceSize(t *testing.T) {
	pair := testkeys.Generate(t)

	encryptedData, err := Encrypt(
		[]byte(`{"id":"Alloc","type":"gauge","value":5.11}`),
		pair.PublicKey,
	)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	var env envelope
	if err := json.Unmarshal(encryptedData, &env); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	wantNonceSize := len(env.Nonce)
	env.Nonce = env.Nonce[:wantNonceSize-1]

	invalidData, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	_, err = Decrypt(invalidData, pair.PrivateKey)
	if err == nil {
		t.Fatal("Decrypt() error = nil, want error")
	}

	got := err.Error()
	want := fmt.Sprintf(
		"invalid nonce size: got %d, want %d",
		len(env.Nonce),
		wantNonceSize,
	)
	if got != want {
		t.Errorf("Decrypt() error = %q, want %q", got, want)
	}
}
