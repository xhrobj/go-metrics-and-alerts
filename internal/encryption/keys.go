package encryption

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	// прочитать файл
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	// декодировать pem
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode public key PEM")
	}

	// попробовать PKIX
	if publicKey, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not RSA")
		}
		return rsaPublicKey, nil
	}

	// если не вышло - попробовать PKCS#1
	rsaPublicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return rsaPublicKey, nil
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	// прочитать файл
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	// декодировать pem
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode private key PEM")
	}

	// попробовать PKCS#8
	if privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return rsaPrivateKey, nil
	}

	// если не вышло - попробовать более старый PKCS#1
	rsaPrivateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return rsaPrivateKey, nil
}
