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
//   - переменные окружения: ADDRESS, POLL_INTERVAL, REPORT_INTERVAL, RATE_LIMIT, KEY, CRYPTO_KEY, CONFIG
//   - JSON-файл конфигурации
//
// Приоритет источников: env > flag > json > default.
func GetAgentConfig() (AgentConfig, error) {
	return parseAgentConfig(os.Args[1:], os.LookupEnv)
}

func parseAgentConfig(args []string, lookupEnv lookupEnvFunc) (AgentConfig, error) {
	cfg := defaultAgentConfig()
	flagCfg := cfg
	flags := flag.NewFlagSet("agent", flag.ContinueOnError)

	flags.StringVar(&flagCfg.ServerAddr, "a", flagCfg.ServerAddr, "address of the HTTP server (host:port)")
	flags.IntVar(&flagCfg.PollIntervalInSec, "p", flagCfg.PollIntervalInSec, "runtime metrics polling interval in seconds")
	flags.IntVar(&flagCfg.ReportIntervalInSec, "r", flagCfg.ReportIntervalInSec, "metrics reporting interval in seconds")
	flags.IntVar(&flagCfg.RateLimit, "l", flagCfg.RateLimit, "limit of simultaneous outgoing requests")
	flags.StringVar(&flagCfg.Key, "k", flagCfg.Key, "hash key for request signing")
	flags.StringVar(&flagCfg.CryptoKey, "crypto-key", flagCfg.CryptoKey, "path to public crypto key")
	flags.StringVar(&flagCfg.ConfigPath, "c", "", "path to JSON configuration file")
	flags.StringVar(&flagCfg.ConfigPath, "config", "", "path to JSON configuration file")

	if err := flags.Parse(args); err != nil {
		return cfg, err
	}

	setFlags := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	configPath := flagCfg.ConfigPath
	if value, ok := lookupEnv("CONFIG"); ok {
		configPath = value
	}

	cfg.ConfigPath = configPath

	if configPath != "" {
		if err := loadAgentConfigFile(configPath, &cfg); err != nil {
			return cfg, err
		}
	}

	applyAgentFlags(&cfg, flagCfg, setFlags)

	if err := applyAgentEnvironment(&cfg, lookupEnv); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func defaultAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddr:          "localhost:8080",
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}
}

func applyAgentFlags(cfg *AgentConfig, flagCfg AgentConfig, setFlags map[string]bool) {
	if setFlags["a"] {
		cfg.ServerAddr = flagCfg.ServerAddr
	}

	if setFlags["p"] {
		cfg.PollIntervalInSec = flagCfg.PollIntervalInSec
	}

	if setFlags["r"] {
		cfg.ReportIntervalInSec = flagCfg.ReportIntervalInSec
	}

	if setFlags["l"] {
		cfg.RateLimit = flagCfg.RateLimit
	}

	if setFlags["k"] {
		cfg.Key = flagCfg.Key
	}

	if setFlags["crypto-key"] {
		cfg.CryptoKey = flagCfg.CryptoKey
	}
}

func applyAgentEnvironment(cfg *AgentConfig, lookupEnv lookupEnvFunc) error {
	if serverAddr, ok := lookupEnv("ADDRESS"); ok {
		cfg.ServerAddr = serverAddr
	}

	if pollIntervalInSec, ok, err := getEnvInt(lookupEnv, "POLL_INTERVAL"); err != nil {
		return err
	} else if ok {
		cfg.PollIntervalInSec = pollIntervalInSec
	}

	if reportIntervalInSec, ok, err := getEnvInt(lookupEnv, "REPORT_INTERVAL"); err != nil {
		return err
	} else if ok {
		cfg.ReportIntervalInSec = reportIntervalInSec
	}

	if rateLimit, ok, err := getEnvInt(lookupEnv, "RATE_LIMIT"); err != nil {
		return err
	} else if ok {
		cfg.RateLimit = rateLimit
	}

	if key, ok := lookupEnv("KEY"); ok {
		cfg.Key = key
	}

	if cryptoKey, ok := lookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = cryptoKey
	}

	return nil
}
