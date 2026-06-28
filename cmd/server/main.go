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

	"github.com/xhrobj/go-metrics-and-alerts/internal/buildinfo"
	"github.com/xhrobj/go-metrics-and-alerts/internal/logger"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
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
	cfg, err := config.GetServerConfig()
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}

	app, err := server.New(cfg, lg)
	if err != nil {
		return err
	}
	defer app.Close()

	return app.Run(ctx)
}

func printBanner(w io.Writer) error {
	const banner = `
   _____          __         .__                _________
  /     \   _____/  |________|__| ____   ______/   _____/ ______________  __ ___________
 /  \ /  \_/ __ \   __\_  __ \  |/ ___\ /  ___/\_____  \_/ __ \_  __ \  \/ // __ \_  __ \
/    Y    \  ___/|  |  |  | \/  \  \___ \___ \ /        \  ___/|  | \/\   /\  ___/|  | \/
\____|__  /\___  >__|  |__|  |__|\___  >____  >_______  /\___  >__|    \_/  \___  >__|
        \/     \/                    \/     \/        \/     \/                 \/

`
	_, err := io.WriteString(w, banner)

	return err
}
