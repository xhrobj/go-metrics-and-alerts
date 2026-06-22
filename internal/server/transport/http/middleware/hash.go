package middleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/hash"
)

// hashResponseWriter буферизует HTTP-ответ для последующего вычисления хеша
// и добавления заголовка HashSHA256 перед отправкой клиенту.
type hashResponseWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

// newHashResponseWriter создаёт буферизированный HTTP-response writer.
func newHashResponseWriter() *hashResponseWriter {
	return &hashResponseWriter{
		header: make(http.Header),
	}
}

// Header возвращает HTTP-заголовки ответа.
func (w *hashResponseWriter) Header() http.Header {
	return w.header
}

// Write записывает тело ответа во внутренний буфер.
func (w *hashResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.body.Write(p)
}

// WriteHeader сохраняет HTTP-статус ответа.
func (w *hashResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
}

// WithHash проверяет подпись входящего HTTP-запроса и добавляет подпись
// в исходящий HTTP-ответ.
//
// Если hashKey не задан, middleware не выполняет проверку и не подписывает ответ.
//
// Для входящего запроса:
//   - если заголовок HashSHA256 присутствует, middleware вычисляет хеш тела
//     запроса с учётом ключа и сравнивает его со значением заголовка;
//   - при несовпадении сервер возвращает status 400 Bad Request;
//   - после чтения тело запроса восстанавливается, чтобы его могли прочитать
//     следующие middleware и handlers.
//
// Для исходящего ответа:
//   - middleware буферизует тело ответа;
//   - после формирования ответа вычисляет хеш и добавляет заголовок
//     HashSHA256 перед отправкой клиенту.
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

				if expectedHash != "" {
					actualHash := hash.CalcHash(body, hashKey)

					if expectedHash != actualHash {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}

				r.Body = io.NopCloser(bytes.NewReader(body))
			}

			// буферизуем ответ, чтобы подписать его после формирования
			hw := newHashResponseWriter()
			next.ServeHTTP(hw, r)

			responseBody := hw.body.Bytes()
			responseHash := hash.CalcHash(responseBody, hashKey)
			if responseHash != "" {
				hw.header.Set("HashSHA256", responseHash)
			}

			// копируем заголовки
			for k, values := range hw.header {
				for _, v := range values {
					w.Header().Add(k, v)
				}
			}

			statusCode := hw.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			w.WriteHeader(statusCode)

			if len(responseBody) > 0 {
				_, _ = w.Write(responseBody)
			}
		})
	}
}
