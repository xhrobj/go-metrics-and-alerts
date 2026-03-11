package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	appmiddleware "github.com/xhrobj/go-metrics-and-alerts/internal/middleware"
	"go.uber.org/zap"
)

func New(h *handler.Handler, log *zap.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.StripSlashes)
	r.Use(appmiddleware.WithLogging(log))
	r.Use(appmiddleware.WithGzip)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("pong"))
	})

	r.Post("/update", h.UpdateJSON)
	r.HandleFunc("/update/{type}/{name}/{value}", h.Update)

	r.Get("/value/{type}/{name}", h.Value)
	r.Post("/value", h.ValueJSON)

	r.Get("/", h.Index)

	return r
}
