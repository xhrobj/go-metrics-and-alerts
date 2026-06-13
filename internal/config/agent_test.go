package config

import "testing"

func TestParseAgentConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want AgentConfig
	}{
		{
			name: "defaults",
			want: AgentConfig{
				ServerAddr:          "localhost:8080",
				PollIntervalInSec:   2,
				ReportIntervalInSec: 10,
				RateLimit:           5,
			},
		},
		{
			name: "flags",
			args: []string{
				"-a", "agent:8081",
				"-p", "3",
				"-r", "11",
				"-l", "7",
				"-k", "flag-key",
				"--crypto-key", "public.pem",
			},
			want: AgentConfig{
				ServerAddr:          "agent:8081",
				PollIntervalInSec:   3,
				ReportIntervalInSec: 11,
				RateLimit:           7,
				Key:                 "flag-key",
				CryptoKey:           "public.pem",
			},
		},
		{
			name: "environment overrides flags",
			args: []string{
				"-a", "flag-agent:8081",
				"-p", "3",
				"-r", "11",
				"-l", "7",
				"-k", "flag-key",
				"--crypto-key", "flag-public.pem",
			},
			env: map[string]string{
				"ADDRESS":         "env-agent:8082",
				"POLL_INTERVAL":   "4",
				"REPORT_INTERVAL": "12",
				"RATE_LIMIT":      "8",
				"KEY":             "env-key",
				"CRYPTO_KEY":      "env-public.pem",
			},
			want: AgentConfig{
				ServerAddr:          "env-agent:8082",
				PollIntervalInSec:   4,
				ReportIntervalInSec: 12,
				RateLimit:           8,
				Key:                 "env-key",
				CryptoKey:           "env-public.pem",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentConfig(tt.args, testLookupEnv(tt.env))
			if err != nil {
				t.Fatalf("parseAgentConfig() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("parseAgentConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseAgentConfigInvalidEnvironment(t *testing.T) {
	_, err := parseAgentConfig(nil, testLookupEnv(map[string]string{
		"POLL_INTERVAL": "invalid",
	}))
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}
}
