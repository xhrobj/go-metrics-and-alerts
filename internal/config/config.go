package config

import (
	"flag"

	serverConfig "github.com/xhrobj/go-metrics-and-alerts/internal/handler/config"
)

type Config struct {
	ServerConfig serverConfig.Config
}

func GetConfig() Config {
	cfg := Config{
		ServerConfig: serverConfig.Config{},
	}

	flag.StringVar(&cfg.ServerConfig.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()

	return cfg
}
