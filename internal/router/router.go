package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	appmiddleware "github.com/xhrobj/go-metrics-and-alerts/internal/middleware"
	"go.uber.org/zap"
)

// New создаёт и настраивает HTTP-роутер:
// регистрирует маршруты и подключает middleware.
func New(h *handler.Handler, log *zap.Logger, hashKey string) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.StripSlashes)
	r.Use(appmiddleware.WithLogging(log))
	r.Use(appmiddleware.WithHash(hashKey))
	r.Use(appmiddleware.WithGzip)

	r.Get("/ping", h.Ping)

	r.HandleFunc("/update/{type}/{name}/{value}", h.Update)
	r.Post("/updates", h.UpdatesJSON)
	r.Post("/update", h.UpdateJSON)

	r.Get("/value/{type}/{name}", h.Value)
	r.Post("/value", h.ValueJSON)

	r.Get("/", h.Index)

	return r
}
