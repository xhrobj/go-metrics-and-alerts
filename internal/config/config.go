package config

import (
	"flag"

	agentConfig "github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	serverConfig "github.com/xhrobj/go-metrics-and-alerts/internal/handler/config"
)

func GetAgentConfig() agentConfig.Config {
	cfg := agentConfig.Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of the HTTP server (host:port)")
	flag.IntVar(&cfg.PollIntervalInSec, "p", 2, "runtime metrics polling interval in seconds")
	flag.IntVar(&cfg.ReportIntervalInSec, "r", 10, "metrics reporting interval in seconds")

	flag.Parse()

	return cfg
}

func GetServerConfig() serverConfig.Config {
	cfg := serverConfig.Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")

	flag.Parse()

	return cfg
}
