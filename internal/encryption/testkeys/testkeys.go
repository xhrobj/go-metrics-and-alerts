package testkeys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

type Pair struct {
	PrivateKey     *rsa.PrivateKey
	PublicKey      *rsa.PublicKey
	PrivateKeyPath string
	PublicKeyPath  string
}

// Generate создает временную RSA-пару для (only!) тестов.
//
// Публичный ключ записывается в формате PKIX,
// приватный ключ - в формате PKCS#8.
func Generate(t testing.TB) Pair {
	t.Helper()

	const rsaKeyBits = 2048

	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	tempDir := t.TempDir()

	publicKeyPath := filepath.Join(tempDir, "public.pem")
	writePublicKey(t, publicKeyPath, &privateKey.PublicKey)

	privateKeyPath := filepath.Join(tempDir, "private.pem")
	writePrivateKey(t, privateKeyPath, privateKey)

	return Pair{
		PrivateKey:     privateKey,
		PublicKey:      &privateKey.PublicKey,
		PrivateKeyPath: privateKeyPath,
		PublicKeyPath:  publicKeyPath,
	}
}

func writePublicKey(
	t testing.TB,
	path string,
	publicKey *rsa.PublicKey,
) {
	t.Helper()

	data, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey() error = %v", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: data,
	})

	if err := os.WriteFile(path, pemData, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func writePrivateKey(
	t testing.TB,
	path string,
	privateKey *rsa.PrivateKey,
) {
	t.Helper()

	data, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error = %v", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: data,
	})

	if err := os.WriteFile(path, pemData, 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}
