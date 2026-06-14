package config

import (
	"flag"
	"os"
)

// AgentConfig содержит параметры конфигурации Агента.
type AgentConfig struct {
	// ServerAddr - адрес и порт HTTP-сервера сбора метрик.
	ServerAddr string

	// PollIntervalInSec - интервал опроса runtime-метрик в секундах.
	PollIntervalInSec int

	// ReportIntervalInSec - интервал отправки метрик на сервер в секундах.
	ReportIntervalInSec int

	// RateLimit - максимальное количество одновременно исходящих запросов от Агента к Серверу.
	RateLimit int

	// Key - "секретный" ключ для вычисления и проверки подписи HTTP-запросов/ответов.
	// Если не задан, подпись не используется.
	Key string

	// CryptoKey - путь к файлу публичного ключа для шифрования запросов.
	// Если не задан, шифрование не используется.
	CryptoKey string

	// ConfigPath - путь к JSON-файлу конфигурации.
	ConfigPath string
}

// GetAgentConfig возвращает конфигурацию агента.
//
// Значения параметров могут быть заданы через:
//   - флаги: -a -p -r -l -k --crypto-key -c/--config
//   - переменные окружения: ADDRESS, POLL_INTERVAL и REPORT_INTERVAL, RATE_LIMIT, KEY, CRYPTO_KEY, CONFIG
//
// Приоритет источников: env > flag > json > default.
func GetAgentConfig() (AgentConfig, error) {
	return parseAgentConfig(os.Args[1:], os.LookupEnv)
}

func parseAgentConfig(args []string, lookupEnv lookupEnvFunc) (AgentConfig, error) {
	cfg := AgentConfig{}
	flags := flag.NewFlagSet("agent", flag.ContinueOnError)

	flags.StringVar(&cfg.ServerAddr, "a", "localhost:8080", "address of the HTTP server (host:port)")
	flags.IntVar(&cfg.PollIntervalInSec, "p", 2, "runtime metrics polling interval in seconds")
	flags.IntVar(&cfg.ReportIntervalInSec, "r", 10, "metrics reporting interval in seconds")
	flags.IntVar(&cfg.RateLimit, "l", 5, "limit of simultaneous outgoing requests")
	flags.StringVar(&cfg.Key, "k", "", "hash key for request signing")
	flags.StringVar(&cfg.CryptoKey, "crypto-key", "", "path to public crypto key")
	flags.StringVar(&cfg.ConfigPath, "c", "", "path to JSON configuration file")
	flags.StringVar(&cfg.ConfigPath, "config", "", "path to JSON configuration file")

	if err := flags.Parse(args); err != nil {
		return cfg, err
	}

	if serverAddr, ok := lookupEnv("ADDRESS"); ok {
		cfg.ServerAddr = serverAddr
	}

	if pollIntervalInSec, ok, err := getEnvInt(lookupEnv, "POLL_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.PollIntervalInSec = pollIntervalInSec
	}

	if reportIntervalInSec, ok, err := getEnvInt(lookupEnv, "REPORT_INTERVAL"); err != nil {
		return cfg, err
	} else if ok {
		cfg.ReportIntervalInSec = reportIntervalInSec
	}

	if rateLimit, ok, err := getEnvInt(lookupEnv, "RATE_LIMIT"); err != nil {
		return cfg, err
	} else if ok {
		cfg.RateLimit = rateLimit
	}

	if key, ok := lookupEnv("KEY"); ok {
		cfg.Key = key
	}

	if cryptoKey, ok := lookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = cryptoKey
	}

	if configPath, ok := lookupEnv("CONFIG"); ok {
		cfg.ConfigPath = configPath
	}

	return cfg, nil
}
