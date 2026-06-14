package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
)

func WithDecryption(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			algorithm := r.Header.Get(encryption.HeaderContentEncryption)
			if algorithm == "" {
				next.ServeHTTP(w, r)
				return
			}

			encryptedBody, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			decryptedBody, err := encryption.Decrypt(encryptedBody, privateKey)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decryptedBody))
			r.ContentLength = int64(len(decryptedBody))
			r.Header.Del(encryption.HeaderContentEncryption)

			next.ServeHTTP(w, r)
		})
	}
}
