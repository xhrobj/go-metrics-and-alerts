package router

import (
	"crypto/rsa"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	appmiddleware "github.com/xhrobj/go-metrics-and-alerts/internal/middleware"
	"go.uber.org/zap"
)

// Options содержит настройки HTTP-роутера.
type Options struct {
	HashKey       string
	PrivateKey    *rsa.PrivateKey
	TrustedSubnet *net.IPNet
}

// New создаёт и настраивает HTTP-роутер:
// регистрирует маршруты и подключает middleware.
func New(h *handler.Handler, log *zap.Logger, opts Options) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.StripSlashes)
	r.Use(appmiddleware.WithLogging(log))

	// logging -> hash -> decryption -> gzip -> [trusted subnet] -> handler

	if opts.HashKey != "" {
		r.Use(appmiddleware.WithHash(opts.HashKey))
	}

	if opts.PrivateKey != nil {
		r.Use(appmiddleware.WithDecryption(opts.PrivateKey))
	}

	r.Use(appmiddleware.WithGzip)

	r.Get("/ping", h.Ping)

	// NOTE: доверенную сеть проверяем только при отправке метрик Агентом Серверу (см. С9И27)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.WithTrustedSubnet(opts.TrustedSubnet))

		r.HandleFunc("/update/{type}/{name}/{value}", h.Update)
		r.Post("/updates", h.UpdatesJSON)
		r.Post("/update", h.UpdateJSON)
	})

	r.Get("/value/{type}/{name}", h.Value)
	r.Post("/value", h.ValueJSON)

	r.Get("/", h.Index)

	return r
}
