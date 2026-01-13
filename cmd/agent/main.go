package main

import (
	"log"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent"
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := config.GetAgentConfig()

	repo := repository.NewMemStorage()
	a, err := agent.New(
		repo,
		cfg.ServerAddr,
		cfg.PollIntervalInSec,
		cfg.ReportIntervalInSec,
	)

	if err != nil {
		return err
	}

	a.Run()

	return nil
}
