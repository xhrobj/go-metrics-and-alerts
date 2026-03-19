package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

func CalcHash(body []byte, key string) string {
	if key == "" {
		return ""
	}

	data := append(body, []byte(key)...)
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
