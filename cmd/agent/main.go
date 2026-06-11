package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent"
	"github.com/xhrobj/go-metrics-and-alerts/internal/buildinfo"
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	if err := buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit); err != nil {
		log.Fatal(err)
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.GetAgentConfig()
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}

	repo := repository.NewMemStorage()
	a, err := agent.New(repo, cfg, lg)

	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	a.Run(ctx)

	return nil
}
