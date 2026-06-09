package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressWriter — обёртка над http.ResponseWriter, которая при необходимости
// сжимает тело HTTP-ответа с помощью gzip
//
// реальный writer выбирается при первой записи ответа
// (в Write или WriteHeader) на основе заголовка Content-Type
//
// Если ответ имеет тип application/json или text/html,
// включается gzip-сжатие. В остальных случаях данные
// записываются напрямую в исходный ResponseWriter
//
// Поля структуры:
//
//	w      — исходный ResponseWriter (используется без сжатия)
//	zw     — gzip.Writer, создаётся только при включении gzip
//	writer — активный writer для тела ответа (либо zw, либо w)
type compressWriter struct {
	w      http.ResponseWriter
	zw     *gzip.Writer
	writer io.Writer
}

// newCompressWriter создаёт новый compressWriter
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w: w,
	}
}

// Header возвращает HTTP-заголовки ответа
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Write записывает тело ответа
//
// если заголовки ещё не были отправлены, перед первой записью
// автоматически отправляется статус 200
func (c *compressWriter) Write(p []byte) (int, error) {
	if c.writer == nil {
		c.WriteHeader(http.StatusOK)
	}
	return c.writer.Write(p)
}

// WriteHeader устанавливает HTTP-код ответа
//
// перед отправкой статуса и заголовков выбирается способ записи
// тела ответа: с gzip-сжатием или без него
func (c *compressWriter) WriteHeader(statusCode int) {
	if c.writer == nil {
		c.initWriter()
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer, если сжатие использовалось
//
// необходимо, чтобы gzip отправил оставшиеся данные
// из внутреннего буфера в ResponseWriter
func (c *compressWriter) Close() error {
	if c.zw != nil {
		return c.zw.Close()
	}
	return nil
}

// compressReader — обёртка над io.ReadCloser,
// которая прозрачно распаковывает gzip-сжатое тело запроса
//
// используется, если клиент прислал запрос с заголовком:
// Content-Encoding: gzip
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// newCompressReader создаёт compressReader
// для чтения gzip-сжатого тела HTTP-запроса
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

// Read читает распакованные данные из тела запроса
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
			cw := newCompressWriter(w)
			ow = cw

			// После выполнения хендлера нужно закрыть gzip.Writer,
			// чтобы отправить все данные из буфера
			defer func() {
				_ = cw.Close()
			}()
		}

		// проверяем, прислал ли клиент сжатое тело запроса
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(strings.ToLower(contentEncoding), "gzip")

		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			r.Body = cr

			defer func() {
				_ = cr.Close()
			}()
		}

		// передаём управление следующему хендлеру
		h.ServeHTTP(ow, r)
	})
}

// initWriter выбирает способ записи тела ответа
//
// если Content-Type ответа — application/json или text/html,
// включается gzip-сжатие. В остальных случаях данные
// записываются напрямую в исходный ResponseWriter
func (c *compressWriter) initWriter() {
	if c.writer != nil {
		return
	}

	contentType := strings.ToLower(c.Header().Get("Content-Type"))

	if strings.HasPrefix(contentType, "application/json") ||
		strings.HasPrefix(contentType, "text/html") {
		c.Header().Set("Content-Encoding", "gzip")
		c.zw = gzip.NewWriter(c.w)
		c.writer = c.zw
		return
	}

	c.writer = c.w
}
