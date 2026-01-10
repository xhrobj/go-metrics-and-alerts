package main

import (
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	repo := repository.NewMemStorage()
	h := handler.New(repo)
	r := router.New(h)

	return http.ListenAndServe(`localhost:8080`, r)
}
