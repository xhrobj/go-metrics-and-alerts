package config

import (
	"flag"
	"os"
)

// AgentConfig содержит параметры конфигурации Агента.
type AgentConfig struct {
	// ServerAddr - адрес и порт Сервера для выбранного транспорта.
	ServerAddr string

	// Transport - транспорт отправки метрик: grpc или http.
	Transport Transport

	// PollIntervalInSec - интервал опроса runtime-метрик в секундах.
	PollIntervalInSec int

	// ReportIntervalInSec - интервал отправки метрик на сервер в секундах.
	ReportIntervalInSec int

	// RateLimit - максимальное количество одновременно исходящих запросов от Агента к Серверу.
	RateLimit int

	// Key - "секретный" ключ для подписи HTTP-запросов.
	// В gRPC-режиме не используется.
	Key string

	// CryptoKey - путь к публичному RSA-ключу для шифрования HTTP-запросов.
	// В gRPC-режиме не используется.
	CryptoKey string

	// GRPCTLSCA - путь к сертификату центра сертификации,
	// которым подписан сертификат gRPC-Сервера.
	// Если не задан, gRPC-соединение устанавливается без TLS.
	GRPCTLSCA string
}

// GetAgentConfig возвращает конфигурацию Агента.
//
// Значения параметров могут быть заданы через:
//   - флаги: -a --transport -p -r -l -k --crypto-key --grpc-tls-ca -c/--config
//   - переменные окружения: ADDRESS, TRANSPORT, POLL_INTERVAL, REPORT_INTERVAL,
//     RATE_LIMIT, KEY, CRYPTO_KEY, GRPC_TLS_CA, CONFIG
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

	flags.StringVar(&flagCfg.ServerAddr, "a", flagCfg.ServerAddr, "address of the Server for selected transport (host:port)")

	transport := string(flagCfg.Transport)
	flags.StringVar(&transport, "transport", transport, "metrics transport: grpc or http")

	flags.IntVar(&flagCfg.PollIntervalInSec, "p", flagCfg.PollIntervalInSec, "runtime metrics polling interval in seconds")
	flags.IntVar(&flagCfg.ReportIntervalInSec, "r", flagCfg.ReportIntervalInSec, "metrics reporting interval in seconds")
	flags.IntVar(&flagCfg.RateLimit, "l", flagCfg.RateLimit, "limit of simultaneous outgoing requests")
	flags.StringVar(&flagCfg.Key, "k", flagCfg.Key, "hash key for request signing")
	flags.StringVar(&flagCfg.CryptoKey, "crypto-key", flagCfg.CryptoKey, "path to public crypto key")
	flags.StringVar(&flagCfg.GRPCTLSCA, "grpc-tls-ca", flagCfg.GRPCTLSCA, "path to gRPC TLS CA certificate")

	var configPath string
	flags.StringVar(&configPath, "c", "", "path to JSON configuration file")
	flags.StringVar(&configPath, "config", "", "path to JSON configuration file")

	if err := flags.Parse(args); err != nil {
		return cfg, err
	}

	flagCfg.Transport = Transport(transport)

	setFlags := make(map[string]bool)
	flags.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if value, ok := lookupEnv("CONFIG"); ok {
		configPath = value
	}

	if configPath != "" {
		if err := loadAgentConfigFile(configPath, &cfg); err != nil {
			return cfg, err
		}
	}

	applyAgentFlags(&cfg, flagCfg, setFlags)

	if err := applyAgentEnvironment(&cfg, lookupEnv); err != nil {
		return cfg, err
	}

	if err := validateAgentConfig(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func defaultAgentConfig() AgentConfig {
	return AgentConfig{
		ServerAddr:          "localhost:50051",
		Transport:           TransportGRPC,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}
}

func applyAgentFlags(cfg *AgentConfig, flagCfg AgentConfig, setFlags map[string]bool) {
	if setFlags["a"] {
		cfg.ServerAddr = flagCfg.ServerAddr
	}

	if setFlags["transport"] {
		cfg.Transport = flagCfg.Transport
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

	if setFlags["grpc-tls-ca"] {
		cfg.GRPCTLSCA = flagCfg.GRPCTLSCA
	}
}

func applyAgentEnvironment(cfg *AgentConfig, lookupEnv lookupEnvFunc) error {
	if serverAddr, ok := lookupEnv("ADDRESS"); ok {
		cfg.ServerAddr = serverAddr
	}

	if transport, ok := lookupEnv("TRANSPORT"); ok {
		cfg.Transport = Transport(transport)
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

	if grpcTLSCA, ok := lookupEnv("GRPC_TLS_CA"); ok {
		cfg.GRPCTLSCA = grpcTLSCA
	}

	return nil
}
