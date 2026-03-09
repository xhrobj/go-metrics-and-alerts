package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/middleware"
	"go.uber.org/zap"
)

func New(h *handler.Handler, log *zap.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.WithLogging(log))

	r.Post("/update", h.UpdateJSON)
	r.HandleFunc("/update/{type}/{name}/{value}", h.Update)
	r.Get("/value/{type}/{name}", h.Value)
	r.Get("/", h.Index)

	return r
}
