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

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
	defer stop()

	if err := run(ctx); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.GetAgentConfig()
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}

	sender, err := agent.NewHTTPSender(cfg.ServerAddr, cfg.Key, cfg.CryptoKey)
	if err != nil {
		return err
	}

	repo := repository.NewMemStorage()
	service := agent.NewReportingService(repo, sender, lg)

	a, err := agent.New(service, cfg, lg)
	if err != nil {
		return err
	}

	return a.Run(ctx)
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
