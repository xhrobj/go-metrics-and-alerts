package main

import (
	"log"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetServerConfig()

	repo := repository.NewMemStorage()
	h := handler.New(repo)
	r := router.New(h)

	log.Printf("running server on %s ...", cfg.ServerAddr)

	return http.ListenAndServe(cfg.ServerAddr, r)
}
