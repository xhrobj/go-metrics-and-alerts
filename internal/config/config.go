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
// Значения параметров могут быть заданы через:
//   - флаги: -a -p -r -l -k
//   - переменные окружения: ADDRESS, POLL_INTERVAL и REPORT_INTERVAL, RATE_LIMIT, KEY
//
// Приоритет источников: env > flag > default.
func GetAgentConfig() (agentConfig.Config, error) {
	cfg := agentConfig.Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of the HTTP server (host:port)")
	flag.IntVar(&cfg.PollIntervalInSec, "p", 2, "runtime metrics polling interval in seconds")
	flag.IntVar(&cfg.ReportIntervalInSec, "r", 10, "metrics reporting interval in seconds")
	flag.IntVar(&cfg.RateLimit, "l", 5, "limit of simultaneous outgoing requests")
	flag.StringVar(&cfg.Key, "k", "", "hash key for request signing")

	flag.Parse()

	if serverAddr, ok := os.LookupEnv("ADDRESS"); ok {
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

	if rateLimit, ok, err := getEnvInt("RATE_LIMIT"); err != nil {
		return cfg, err
	} else if ok {
		cfg.RateLimit = rateLimit
	}

	if key, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = key
	}

	return cfg, nil
}

// GetServerConfig возвращает конфигурацию HTTP-сервера.
//
// Значения параметров могут быть заданы через:
//   - флаги: -a -i -f -r -d -k
//   - переменные окружения: ADDRESS, STORE_INTERVAL, FILE_STORAGE_PATH, RESTORE, DATABASE_DSN, KEY
//
// Приоритет источников: env > flag > default.
func GetServerConfig() (serverConfig.Config, error) {
	cfg := serverConfig.Config{}

	flag.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address and port to run server")
	flag.IntVar(&cfg.StoreIntervalInSec, "i", 300, "store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", "metrics-db.json", "path to metrics storage file")
	flag.BoolVar(&cfg.Restore, "r", false, "restore metrics from file on startup")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")
	flag.StringVar(&cfg.Key, "k", "", "hash key for request signing")

	flag.Parse()

	if serverAddr, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.ServerAddr = serverAddr
	}

	if storeIntervalInSec, ok, err := getEnvInt("STORE_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.StoreIntervalInSec = storeIntervalInSec
	}

	if fileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = fileStoragePath
	}

	if restore, ok, err := getEnvBool("RESTORE"); err != nil {
		return cfg, err
	} else if ok {
		cfg.Restore = restore
	}

	if databaseDSN, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = databaseDSN
	}

	if key, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = key
	}

	return cfg, nil
}

func getEnvInt(name string) (int, bool, error) {
	if v, ok := os.LookupEnv(name); ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, false, err
		}
		return i, true, nil
	}
	return 0, false, nil
}

func getEnvBool(name string) (bool, bool, error) {
	if v, ok := os.LookupEnv(name); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return false, false, err
		}
		return b, true, nil
	}
	return false, false, nil
}
