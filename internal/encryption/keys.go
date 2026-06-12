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
		return nil, fmt.Errorf("read public key: %w", err)
	}

	// декодировать pem
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode public key PEM")
	}

	// попробовать PKCS#1
	if privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return privateKey, nil
	}

	// если не вышло - попробовать PKCS#8
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	// проверить что это rsa
	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	return rsaPrivateKey, nil
}
