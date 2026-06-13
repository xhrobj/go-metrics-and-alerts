package main

import (
	"context"
	"errors"
	"flag"
	"io"
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
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	if err := buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit); err != nil {
		log.Fatal(err)
	}

	if err := printBanner(os.Stdout); err != nil {
		log.Fatal(err)
	}

	if err := run(); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

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

func printBanner(w io.Writer) error {
	const banner = `
   _____          __         .__                 _____                         __
  /     \   _____/  |________|__| ____   ______ /  _  \    ____   ____   _____/  |_
 /  \ /  \_/ __ \   __\_  __ \  |/ ___\ /  ___//  /_\  \  / ___\_/ __ \ /    \   __\
/    Y    \  ___/|  |  |  | \/  \  \___ \___ \/    |    \/ /_/  >  ___/|   |  \  |
\____|__  /\___  >__|  |__|  |__|\___  >____  >____|__  /\___  / \___  >___|  /__|
        \/     \/                    \/     \/        \//_____/      \/     \/

`
	_, err := io.WriteString(w, banner)

	return err
}
