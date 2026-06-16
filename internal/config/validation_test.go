package config

import (
	"strings"
	"testing"
)

func TestValidateAgentConfig(t *testing.T) {
	valid := defaultAgentConfig()

	tests := []struct {
		name      string
		change    func(*AgentConfig)
		wantError string
	}{
		{
			name: "valid",
		},
		{
			name: "empty server address",
			change: func(cfg *AgentConfig) {
				cfg.ServerAddr = ""
			},
			wantError: "server address",
		},
		{
			name: "invalid poll interval",
			change: func(cfg *AgentConfig) {
				cfg.PollIntervalInSec = 0
			},
			wantError: "poll interval",
		},
		{
			name: "invalid report interval",
			change: func(cfg *AgentConfig) {
				cfg.ReportIntervalInSec = 0
			},
			wantError: "report interval",
		},
		{
			name: "invalid rate limit",
			change: func(cfg *AgentConfig) {
				cfg.RateLimit = 0
			},
			wantError: "rate limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			if tt.change != nil {
				tt.change(&cfg)
			}

			err := validateAgentConfig(cfg)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateAgentConfig() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateAgentConfig() error = nil, want %q", tt.wantError)
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateAgentConfig() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestValidateServerConfig(t *testing.T) {
	valid := defaultServerConfig()

	tests := []struct {
		name      string
		change    func(*ServerConfig)
		wantError string
	}{
		{
			name: "valid",
		},
		{
			name: "empty server address",
			change: func(cfg *ServerConfig) {
				cfg.ServerAddr = ""
			},
			wantError: "server address",
		},
		{
			name: "negative store interval",
			change: func(cfg *ServerConfig) {
				cfg.StoreIntervalInSec = -1
			},
			wantError: "store interval",
		},
		{
			name: "invalid trusted subnet",
			change: func(cfg *ServerConfig) {
				cfg.TrustedSubnet = "192.168.1.0/999"
			},
			wantError: "trusted subnet",
		},
		{
			name: "invalid audit URL",
			change: func(cfg *ServerConfig) {
				cfg.AuditURL = "://bad-url"
			},
			wantError: "audit URL",
		},
		{
			name: "unsupported audit URL scheme",
			change: func(cfg *ServerConfig) {
				cfg.AuditURL = "ftp://audit.example.com"
			},
			wantError: "http or https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := valid
			if tt.change != nil {
				tt.change(&cfg)
			}

			err := validateServerConfig(cfg)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateServerConfig() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateServerConfig() error = nil, want %q", tt.wantError)
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateServerConfig() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestParseAgentConfigValidatesResult(t *testing.T) {
	_, err := parseAgentConfig(
		[]string{"-p", "0"},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}
}

func TestParseServerConfigValidatesResult(t *testing.T) {
	_, err := parseServerConfig(
		[]string{"-t", "invalid"},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}
