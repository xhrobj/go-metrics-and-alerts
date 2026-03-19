package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/hash"
)

// WithHash проверяет подпись входящего HTTP-запроса.
//
//   - если заголовок HashSHA256 присутствует, middleware вычисляет хеш тела
//     запроса с учётом ключа и сравнивает его со значением заголовка;
//   - при несовпадении сервер возвращает status 400 Bad Request;
//   - после чтения тело запроса восстанавливается, чтобы его могли прочитать
//     следующие middleware и handlers.
func WithHash(hashKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hashKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			if r.Body != nil {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				expectedHash := r.Header.Get("HashSHA256")

				// NOTE: индексная страница из браузера не шлёт нам заголовок с хешем
				if expectedHash != "" {
					actualHash := hash.CalcHash(body, hashKey)

					fmt.Println("expected:", expectedHash)
					fmt.Println("actual:", actualHash)

					if expectedHash != actualHash {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
			}
		})
	}
}
