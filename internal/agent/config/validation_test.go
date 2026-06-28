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
			name: "invalid transport",
			change: func(cfg *AgentConfig) {
				cfg.Transport = "unknown"
			},
			wantError: "transport",
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

			assertValidationResult(t, validateAgentConfig(cfg), tt.wantError)
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

func assertValidationResult(t *testing.T, err error, wantError string) {
	t.Helper()

	switch {
	case wantError == "" && err != nil:
		t.Fatalf("validation error = %v, want nil", err)
	case wantError != "" && err == nil:
		t.Fatalf("validation error = nil, want containing %q", wantError)
	case wantError != "" && !strings.Contains(err.Error(), wantError):
		t.Fatalf("validation error = %q, want substring %q", err, wantError)
	}
}
