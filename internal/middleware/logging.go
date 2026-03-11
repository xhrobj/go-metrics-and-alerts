package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// loggingResponseWriter — обёртка над http.ResponseWriter,
// которая перехватывает код статуса ответа и размер записанного тела.
// Используется middleware для логирования сведений об ответе.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// newLoggingResponseWriter создаёт loggingResponseWriter
// вокруг стандартного http.ResponseWriter.
//
// По умолчанию устанавливает статус http.StatusOK,
// поскольку если хендлер не вызывает WriteHeader,
// HTTP считает ответ успешным (200 OK).
func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader перехватывает установку HTTP-статуса ответа,
// сохраняет код статуса и передаёт вызов исходному ResponseWriter.
func (lw *loggingResponseWriter) WriteHeader(statusCode int) {
	lw.statusCode = statusCode
	lw.ResponseWriter.WriteHeader(statusCode)
}

// Write перехватывает запись тела ответа.
// Метод сохраняет количество записанных байт и передаёт запись исходному ResponseWriter.
func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lw.ResponseWriter.Write(b)
	// увеличивает счётчик size
	lw.size += n

	return n, err
}

// WithLogging создаёт middleware для логирования HTTP-запросов и ответов.
//
// Middleware фиксирует:
//   - URI запроса
//   - HTTP-метод
//   - время выполнения запроса
//   - код статуса ответа
//   - размер тела ответа
//
// Логирование выполняется с использованием zap.Logger на уровне Info.
func WithLogging(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lw := newLoggingResponseWriter(w)

			next.ServeHTTP(lw, r)

			log.Info("http request completed",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", lw.statusCode),
				zap.Int("size", lw.size),
			)
		})
	}
}
