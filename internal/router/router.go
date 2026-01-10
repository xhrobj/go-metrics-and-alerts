package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
)

func New(h *handler.Handler) http.Handler {
	r := chi.NewRouter()
	r.HandleFunc("/update/*", h.Update)

	return r
}
