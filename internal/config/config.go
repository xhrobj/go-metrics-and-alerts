package config

import (
	"flag"
	"os"
	"strconv"

	agentConfig "github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	serverConfig "github.com/xhrobj/go-metrics-and-alerts/internal/handler/config"
)

// GetAgentConfig возвращает конфигурацию агента.
//
// Значения параметров могут быть заданы через флаги командной строки: -a -p -r
// и переменные окружения: ADDRESS, POLL_INTERVAL и REPORT_INTERVAL.
//
// Приоритет источников: env > flag > default.
func GetAgentConfig() (agentConfig.Config, error) {
	cfg := agentConfig.Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of the HTTP server (host:port)")
	flag.IntVar(&cfg.PollIntervalInSec, "p", 2, "runtime metrics polling interval in seconds")
	flag.IntVar(&cfg.ReportIntervalInSec, "r", 10, "metrics reporting interval in seconds")

	flag.Parse()

	if serverAddr := os.Getenv("ADDRESS"); serverAddr != "" {
		cfg.ServerAddr = serverAddr
	}

	if pollIntervalInSec, ok, err := getEnvInt("POLL_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.PollIntervalInSec = pollIntervalInSec
	}

	if reportIntervalInSec, ok, err := getEnvInt("REPORT_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.ReportIntervalInSec = reportIntervalInSec
	}

	return cfg, nil
}

// GetServerConfig возвращает конфигурацию HTTP-сервера.
//
// Значения параметров могут быть заданы через:
//   - флаг -a
//   - переменную окружения ADDRESS
//
// Приоритет источников: env > flag > default.
func GetServerConfig() serverConfig.Config {
	cfg := serverConfig.Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()

	if serverAddr := os.Getenv("ADDRESS"); serverAddr != "" {
		cfg.ServerAddr = serverAddr
	}

	return cfg
}

func getEnvInt(name string) (int, bool, error) {
	if v := os.Getenv(name); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false, err
		}
		return i, true, nil
	}
	return 0, false, nil
}
