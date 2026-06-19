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

// NOTE: RSA может зашифровать только небольшой объём данных,
// поэтому используется гибридная схема:
// данные шифруются AES-GCM, а AES-ключ - RSA-OAEP.

const (
	// HeaderContentEncryption задаёт имя заголовка со схемой шифрования тела запроса.
	HeaderContentEncryption = "Content-Encryption"

	// SchemeRSAOAEPWithAESGCM обозначает гибридную схему:
	// данные шифруются AES-GCM, а сам AES-ключ - RSA-OAEP.
	SchemeRSAOAEPWithAESGCM = "rsa-aes-gcm"
)

type envelope struct {
	Key   []byte `json:"key"`   // AES-ключ, зашифрованный публичным RSA-ключом
	Nonce []byte `json:"nonce"` // одноразовое значение (number used once) для AES-GCM
	Data  []byte `json:"data"`  // данные, зашифрованные AES-GCM
}

// Encrypt шифрует данные по гибридной схеме:
// данные - с помощью AES-GCM, AES-ключ - с помощью RSA-OAEP.
func Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return nil, fmt.Errorf("public key is nil")
	}

	// генерируем aes-ключ
	const aesKeySize = 32 // AES-256
	aesKey := make([]byte, aesKeySize)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("generate AES key: %w", err)
	}

	// создадем aes-шифр, используя ключ
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	// включаем режим GCM
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

	// упаковываем данные в envelope
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

// Decrypt расшифровывает AES-ключ с помощью RSA-OAEP,
// а затем расшифровывает данные с помощью AES-GCM.
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

	// создаем AES-шифр с полученным ключом
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}

	// включаем режим GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	// длина присланного nonce должна соответствовать размеру, ожидаемому gcm
	if got, want := len(env.Nonce), gcm.NonceSize(); got != want {
		return nil, fmt.Errorf("invalid nonce size: got %d, want %d", got, want)
	}

	// расшифровываем данные
	decryptedData, err := gcm.Open(nil, env.Nonce, env.Data, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt data: %w", err)
	}

	return decryptedData, nil
}
