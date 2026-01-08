package main

import (
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	repo := repository.NewMemStorage()
	h := handler.New(repo)
	mux := newServerRouter(h)

	return http.ListenAndServe(`localhost:8080`, mux)
}

func newServerRouter(h *handler.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", h.Update)

	return mux
}
