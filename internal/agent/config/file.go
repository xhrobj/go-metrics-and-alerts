package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type agentFileConfig struct {
	Address        *string `json:"address"`
	Transport      *string `json:"transport"`
	PollInterval   *string `json:"poll_interval"`
	ReportInterval *string `json:"report_interval"`
	RateLimit      *int    `json:"rate_limit"`
	Key            *string `json:"key"`
	CryptoKey      *string `json:"crypto_key"`
	GRPCTLSCA      *string `json:"grpc_tls_ca"`
}

func loadAgentConfigFile(path string, cfg *AgentConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read agent config file %q: %w", path, err)
	}

	var fileCfg agentFileConfig

	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return fmt.Errorf("decode agent config file %q: %w", path, err)
	}

	if fileCfg.Address != nil {
		cfg.ServerAddr = *fileCfg.Address
	}

	if fileCfg.Transport != nil {
		cfg.Transport = Transport(*fileCfg.Transport)
	}

	if fileCfg.PollInterval != nil {
		pollIntervalInSec, err := parseAgentDurationInSeconds(
			"poll_interval",
			*fileCfg.PollInterval,
		)
		if err != nil {
			return err
		}

		cfg.PollIntervalInSec = pollIntervalInSec
	}

	if fileCfg.ReportInterval != nil {
		reportIntervalInSec, err := parseAgentDurationInSeconds(
			"report_interval",
			*fileCfg.ReportInterval,
		)
		if err != nil {
			return err
		}

		cfg.ReportIntervalInSec = reportIntervalInSec
	}

	if fileCfg.RateLimit != nil {
		cfg.RateLimit = *fileCfg.RateLimit
	}

	if fileCfg.Key != nil {
		cfg.Key = *fileCfg.Key
	}

	if fileCfg.CryptoKey != nil {
		cfg.CryptoKey = *fileCfg.CryptoKey
	}

	if fileCfg.GRPCTLSCA != nil {
		cfg.GRPCTLSCA = *fileCfg.GRPCTLSCA
	}

	return nil
}

func parseAgentDurationInSeconds(name, value string) (int, error) {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s %q: %w", name, value, err)
	}

	if duration%time.Second != 0 {
		return 0, fmt.Errorf(
			"parse %s %q: duration must contain whole seconds",
			name,
			value,
		)
	}

	return int(duration / time.Second), nil
}
