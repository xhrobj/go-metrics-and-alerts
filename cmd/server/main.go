package main

import (
	"fmt"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/handler"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"github.com/xhrobj/go-metrics-and-alerts/internal/router"
)

func main() {
	parseFlags()

	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	repo := repository.NewMemStorage()
	h := handler.New(repo)
	r := router.New(h)

	fmt.Println("*** Running server on", flagRunAddr) // FIXME:

	return http.ListenAndServe(flagRunAddr, r)
}
