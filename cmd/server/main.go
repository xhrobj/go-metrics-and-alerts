package main

import (
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", handler.UpdatePage)
	mux.HandleFunc("/update", handler.UpdatePage) // NOTE: avoid redirect from `/update` -> `/update/`
	return http.ListenAndServe(`localhost:8080`, mux)
}
