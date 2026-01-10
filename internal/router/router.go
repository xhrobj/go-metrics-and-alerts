package router

import (
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
)

func New(h *handler.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", h.Update)

	return mux
}
