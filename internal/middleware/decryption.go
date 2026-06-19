package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"
	"strings"

	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
)

// WithDecryption расшифровывает тело HTTP-запроса приватным RSA-ключом.
//
// Если заголовок Content-Encryption отсутствует, запрос передаётся дальше без изменений.
func WithDecryption(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme := r.Header.Get(encryption.HeaderContentEncryption)
			if scheme == "" {
				next.ServeHTTP(w, r)
				return
			}

			if privateKey == nil ||
				!strings.EqualFold(scheme, encryption.SchemeRSAOAEPWithAESGCM) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if r.Body == nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if err := decryptRequestBody(r, privateKey); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func decryptRequestBody(r *http.Request, privateKey *rsa.PrivateKey) error {
	encryptedBody, err := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if err != nil {
		return err
	}

	decryptedBody, err := encryption.Decrypt(encryptedBody, privateKey)
	if err != nil {
		return err
	}

	r.Body = io.NopCloser(bytes.NewReader(decryptedBody))
	r.ContentLength = int64(len(decryptedBody))
	r.Header.Del(encryption.HeaderContentEncryption)

	return nil
}
