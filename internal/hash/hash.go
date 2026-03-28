package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

// CalcHash вычисляет SHA256-хеш от тела данных с использованием секретного ключа.
//
// Хеш рассчитывается как SHA256(body + key), где key добавляется в конец тела.
// Если ключ пустой, функция возвращает пустую строку, что означает отсутствие подписи.
func CalcHash(body []byte, key string) string {
	if key == "" {
		return ""
	}

	data := make([]byte, 0, len(body)+len(key))
	data = append(data, body...)
	data = append(data, key...)

	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
