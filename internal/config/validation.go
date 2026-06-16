package config

import (
	"fmt"
	"net"
	"net/url"
)

func validateAgentConfig(cfg AgentConfig) error {
	if cfg.ServerAddr == "" {
		return fmt.Errorf("server address must not be empty")
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

func validateServerConfig(cfg ServerConfig) error {
	if cfg.ServerAddr == "" {
		return fmt.Errorf("server address must not be empty")
	}

	if cfg.StoreIntervalInSec < 0 {
		return fmt.Errorf(
			"store interval in seconds must be >= 0, got %d",
			cfg.StoreIntervalInSec,
		)
	}

	if cfg.TrustedSubnet != "" {
		if _, _, err := net.ParseCIDR(cfg.TrustedSubnet); err != nil {
			return fmt.Errorf(
				"parse trusted subnet %q: %w",
				cfg.TrustedSubnet,
				err,
			)
		}
	}

	if cfg.AuditURL != "" {
		auditURL, err := url.ParseRequestURI(cfg.AuditURL)
		if err != nil {
			return fmt.Errorf("parse audit URL %q: %w", cfg.AuditURL, err)
		}

		if auditURL.Scheme != "http" && auditURL.Scheme != "https" {
			return fmt.Errorf(
				"audit URL must use http or https scheme, got %q",
				auditURL.Scheme,
			)
		}

		if auditURL.Host == "" {
			return fmt.Errorf("audit URL must contain host")
		}
	}

	return nil
}
