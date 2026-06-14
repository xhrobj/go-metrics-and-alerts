package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type agentFileConfig struct {
	Address        *string `json:"address"`
	PollInterval   *string `json:"poll_interval"`
	ReportInterval *string `json:"report_interval"`
	RateLimit      *int    `json:"rate_limit"`
	Key            *string `json:"key"`
	CryptoKey      *string `json:"crypto_key"`
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

	if fileCfg.PollInterval != nil {
		pollInterval, err := time.ParseDuration(*fileCfg.PollInterval)
		if err != nil {
			return fmt.Errorf(
				"parse poll_interval %q: %w",
				*fileCfg.PollInterval,
				err,
			)
		}

		if pollInterval%time.Second != 0 {
			return fmt.Errorf(
				"parse poll_interval %q: duration must contain whole seconds",
				*fileCfg.PollInterval,
			)
		}

		cfg.PollIntervalInSec = int(pollInterval / time.Second)
	}

	if fileCfg.ReportInterval != nil {
		reportInterval, err := time.ParseDuration(*fileCfg.ReportInterval)
		if err != nil {
			return fmt.Errorf(
				"parse report_interval %q: %w",
				*fileCfg.ReportInterval,
				err,
			)
		}

		if reportInterval%time.Second != 0 {
			return fmt.Errorf(
				"parse report_interval %q: duration must contain whole seconds",
				*fileCfg.ReportInterval,
			)
		}

		cfg.ReportIntervalInSec = int(reportInterval / time.Second)
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

	return nil
}
