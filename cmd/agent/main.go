package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent"
	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/service"
	grpctransport "github.com/xhrobj/go-metrics-and-alerts/internal/agent/transport/grpc"
	httptransport "github.com/xhrobj/go-metrics-and-alerts/internal/agent/transport/http"
	"github.com/xhrobj/go-metrics-and-alerts/internal/buildinfo"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
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

func run(ctx context.Context) (err error) {
	cfg, err := config.GetAgentConfig()
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}

	sender, closeSender, err := newMetricsSender(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closeSender(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close metrics sender: %w", closeErr))
		}
	}()

	repo := repository.NewMemStorage()
	reportingService := service.New(repo, sender, lg)

	a, err := agent.New(reportingService, cfg, lg)
	if err != nil {
		return err
	}

	logAgentStart(lg, cfg)

	return a.Run(ctx)
}

func logAgentStart(log *zap.Logger, cfg config.AgentConfig) {
	log.Info("running agent",
		zap.String("transport", string(cfg.Transport)),
		zap.String("address", cfg.ServerAddr),
	)
}

func newMetricsSender(
	cfg config.AgentConfig,
) (service.MetricsSender, func() error, error) {
	switch cfg.Transport {
	case config.TransportGRPC:
		sender, err := grpctransport.NewGRPCSender(cfg.ServerAddr)
		if err != nil {
			return nil, nil, err
		}

		return sender, sender.Close, nil

	case config.TransportHTTP:
		sender, err := httptransport.NewHTTPSender(
			cfg.ServerAddr,
			cfg.Key,
			cfg.CryptoKey,
		)
		if err != nil {
			return nil, nil, err
		}

		return sender, func() error { return nil }, nil

	default:
		return nil, nil, fmt.Errorf("unsupported transport %q", cfg.Transport)
	}
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
