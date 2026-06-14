package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
)

// !!!: не получится зашифровать значительнй объем данных чисто rsa -
// наши батчи не пролезут, нужно использовать гибрид rsa+aes ...
//
// https://pkg.go.dev/crypto/rsa
//
// RSA is able to encrypt only a very limited amount of data.
// In order to encrypt reasonable amounts of data a hybrid scheme is commonly used:
// RSA is used to encrypt a key for a symmetric primitive like AES-GCM.

const (
	HeaderContentEncryption = "Content-Encryption"
	SchemeRSAOAEPWithAESGCM = "rsa-aes-gcm"
)

type envelope struct {
	Key   []byte `json:"key"`   // AES-ключ зашифрованный RSA public key'ем
	Nonce []byte `json:"nonce"` // nonce для AES-GCM
	Data  []byte `json:"data"`  // body зашифрованный AES-GCM
}

func Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("public key is nil")
	}

	// генерируем aes-ключ
	aesKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("generate AES key: %w", err)
	}

	// создадем шифр используя ключ
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// генерируем nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// шифруем данные
	encryptedData := gcm.Seal(nil, nonce, data, nil)

	// шифруем aes-ключ
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("encrypt AES key: %w", err)
	}

	// упаковывыем в envelop
	body, err := json.Marshal(envelope{
		Key:   encryptedKey,
		Nonce: nonce,
		Data:  encryptedData,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}

	return body, nil
}

func Decrypt(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}

	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	// расшифровываем aes-ключ из конверта
	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, env.Key, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt AES key: %w", err)
	}

	// создаем шифр с полученным ключом
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	// включаем режим gcm
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// расшифровываем
	decryptedData, err := gcm.Open(nil, env.Nonce, env.Data, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return decryptedData, nil
}
