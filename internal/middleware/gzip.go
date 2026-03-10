package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressWriter — обёртка над http.ResponseWriter,
// которая сжимает данные ответа в формате gzip
//
// middleware подменяет обычный ResponseWriter на compressWriter,
// если клиент указал поддержку gzip через заголовок: Accept-Encoding: gzip
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// newCompressWriter создаёт новый compressWriter
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	w.Header().Set("Content-Encoding", "gzip")

	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Header возвращает HTTP-заголовки ответа
// (из оригинального ResponseWriter'а)
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает данные ответа
func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// WriteHeader устанавливает HTTP-код ответа
func (c *compressWriter) WriteHeader(statusCode int) {
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer
//
// необходимо, чтобы gzip отправил оставшиеся данные
// из внутреннего буфера в ResponseWriter
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// compressReader — обёртка над io.ReadCloser,
// которая прозрачно распаковывает gzip-данные
//
// используется, если клиент прислал запрос с заголовком:
// Content-Encoding: gzip
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// newCompressReader создаёт gzip.Reader
// для распаковки тела HTTP-запроса
func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Read читает распакованные данные из gzip-потока
func (c *compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

// Close закрывает reader и освобождает ресурсы.
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// WithGzip — HTTP middleware, добавляющий поддержку gzip
//
// - распаковывает тело запроса, если клиент прислал: Content-Encoding: gzip
// - cжимает тело ответа, если клиент поддерживает gzip: Accept-Encoding: gzip
func WithGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ow := w

		// проверяем, поддерживает ли клиент gzip-ответы.
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(strings.ToLower(acceptEncoding), "gzip")

		if supportsGzip {

			// TODO: сжимать только json/html

			cw := newCompressWriter(w)
			ow = cw

			// После выполнения хендлера нужно закрыть gzip.Writer,
			// чтобы отправить все данные из буфера
			defer cw.Close()
		}

		// проверяем, прислал ли клиент сжатое тело запроса
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")

		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.Body = cr

			defer cr.Close()
		}

		// передаём управление следующему хендлеру
		h.ServeHTTP(ow, r)
	})
}
