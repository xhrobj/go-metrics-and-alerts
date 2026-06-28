package config

import "fmt"

func validateAgentConfig(cfg AgentConfig) error {
	if cfg.ServerAddr == "" {
		return fmt.Errorf("server address must not be empty")
	}

	switch cfg.Transport {
	case TransportGRPC, TransportHTTP:
	default:
		return fmt.Errorf(
			"transport must be %q or %q, got %q",
			TransportGRPC,
			TransportHTTP,
			cfg.Transport,
		)
	}

	if cfg.PollIntervalInSec <= 0 {
		return fmt.Errorf(
			"poll interval must be > 0, got %d",
			cfg.PollIntervalInSec,
		)
	}

	if cfg.ReportIntervalInSec <= 0 {
		return fmt.Errorf(
			"report interval must be > 0, got %d",
			cfg.ReportIntervalInSec,
		)
	}

	if cfg.RateLimit <= 0 {
		return fmt.Errorf("rate limit must be > 0, got %d", cfg.RateLimit)
	}

	return nil
}
