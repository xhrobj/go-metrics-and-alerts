package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
				ServerAddr:          "localhost:50051",
				Transport:           TransportGRPC,
				PollIntervalInSec:   2,
				ReportIntervalInSec: 10,
				RateLimit:           5,
			},
		},
		{
			name: "flags",
			args: []string{
				"-a", "agent:8081",
				"--transport", "http",
				"-p", "3",
				"-r", "11",
				"-l", "7",
				"-k", "flag-key",
				"--crypto-key", "public.pem",
			},
			want: AgentConfig{
				ServerAddr:          "agent:8081",
				Transport:           TransportHTTP,
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
				"--transport", "grpc",
				"-p", "3",
				"-r", "11",
				"-l", "7",
				"-k", "flag-key",
				"--crypto-key", "flag-public.pem",
			},
			env: map[string]string{
				"ADDRESS":         "env-agent:8082",
				"TRANSPORT":       "http",
				"POLL_INTERVAL":   "4",
				"REPORT_INTERVAL": "12",
				"RATE_LIMIT":      "8",
				"KEY":             "env-key",
				"CRYPTO_KEY":      "env-public.pem",
			},
			want: AgentConfig{
				ServerAddr:          "env-agent:8082",
				Transport:           TransportHTTP,
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

func TestParseAgentConfigFromFile(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{
		"address": "json-agent:8083",
		"transport": "http",
		"poll_interval": "3s",
		"report_interval": "11s",
		"rate_limit": 7,
		"key": "json-key",
		"crypto_key": "json-public.pem"
	}`)

	got, err := parseAgentConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseAgentConfig() error = %v", err)
	}

	want := AgentConfig{
		ServerAddr:          "json-agent:8083",
		Transport:           TransportHTTP,
		PollIntervalInSec:   3,
		ReportIntervalInSec: 11,
		RateLimit:           7,
		Key:                 "json-key",
		CryptoKey:           "json-public.pem",
	}

	if got != want {
		t.Fatalf("parseAgentConfig() = %+v, want %+v", got, want)
	}
}

func TestParseAgentConfigFileKeepsDefaults(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{
		"address": "json-agent:8083"
	}`)

	got, err := parseAgentConfig(
		[]string{"-c", configPath},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseAgentConfig() error = %v", err)
	}

	want := AgentConfig{
		ServerAddr:          "json-agent:8083",
		Transport:           TransportGRPC,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}

	if got != want {
		t.Fatalf("parseAgentConfig() = %+v, want %+v", got, want)
	}
}

func TestParseAgentConfigFlagsOverrideFile(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{
		"address": "json-agent:8083",
		"transport": "grpc",
		"poll_interval": "3s",
		"report_interval": "11s",
		"rate_limit": 7,
		"key": "json-key",
		"crypto_key": "json-public.pem"
	}`)

	got, err := parseAgentConfig(
		[]string{
			"--config", configPath,
			"-a", "flag-agent:8084",
			"--transport", "http",
			"-p", "4",
			"-r", "12",
			"-l", "8",
			"-k", "flag-key",
			"--crypto-key", "flag-public.pem",
		},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseAgentConfig() error = %v", err)
	}

	want := AgentConfig{
		ServerAddr:          "flag-agent:8084",
		Transport:           TransportHTTP,
		PollIntervalInSec:   4,
		ReportIntervalInSec: 12,
		RateLimit:           8,
		Key:                 "flag-key",
		CryptoKey:           "flag-public.pem",
	}

	if got != want {
		t.Fatalf("parseAgentConfig() = %+v, want %+v", got, want)
	}
}

func TestParseAgentConfigEnvironmentOverridesFileAndFlags(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{
		"address": "json-agent:8083",
		"transport": "grpc",
		"poll_interval": "3s",
		"report_interval": "11s",
		"rate_limit": 7,
		"key": "json-key",
		"crypto_key": "json-public.pem"
	}`)

	got, err := parseAgentConfig(
		[]string{
			"--config", configPath,
			"-a", "flag-agent:8084",
			"--transport", "http",
			"-p", "4",
			"-r", "12",
			"-l", "8",
			"-k", "flag-key",
			"--crypto-key", "flag-public.pem",
		},
		testLookupEnv(map[string]string{
			"ADDRESS":         "env-agent:8085",
			"TRANSPORT":       "grpc",
			"POLL_INTERVAL":   "5",
			"REPORT_INTERVAL": "15",
			"RATE_LIMIT":      "9",
			"KEY":             "env-key",
			"CRYPTO_KEY":      "env-public.pem",
		}),
	)
	if err != nil {
		t.Fatalf("parseAgentConfig() error = %v", err)
	}

	want := AgentConfig{
		ServerAddr:          "env-agent:8085",
		Transport:           TransportGRPC,
		PollIntervalInSec:   5,
		ReportIntervalInSec: 15,
		RateLimit:           9,
		Key:                 "env-key",
		CryptoKey:           "env-public.pem",
	}

	if got != want {
		t.Fatalf("parseAgentConfig() = %+v, want %+v", got, want)
	}
}

func TestParseAgentConfigEnvironmentOverridesConfigFlag(t *testing.T) {
	flagConfigPath := writeAgentConfigFile(t, `{
		"address": "flag-file-agent:8081"
	}`)

	envConfigPath := writeAgentConfigFile(t, `{
		"address": "env-file-agent:8082"
	}`)

	got, err := parseAgentConfig(
		[]string{"--config", flagConfigPath},
		testLookupEnv(map[string]string{
			"CONFIG": envConfigPath,
		}),
	)
	if err != nil {
		t.Fatalf("parseAgentConfig() error = %v", err)
	}

	want := AgentConfig{
		ServerAddr:          "env-file-agent:8082",
		Transport:           TransportGRPC,
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}

	if got != want {
		t.Fatalf("parseAgentConfig() = %+v, want %+v", got, want)
	}
}

func TestParseAgentConfigInvalidEnvironment(t *testing.T) {
	_, err := parseAgentConfig(nil, testLookupEnv(map[string]string{
		"POLL_INTERVAL": "invalid",
	}))
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}

	wantError := `POLL_INTERVAL="invalid"`
	if !strings.Contains(err.Error(), wantError) {
		t.Fatalf("parseAgentConfig() error = %q, want substring %q", err, wantError)
	}
}

func TestParseAgentConfigInvalidTransport(t *testing.T) {
	_, err := parseAgentConfig(
		[]string{"--transport", "unknown"},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}

	wantError := `transport must be "grpc" or "http"`
	if !strings.Contains(err.Error(), wantError) {
		t.Fatalf("parseAgentConfig() error = %q, want substring %q", err, wantError)
	}
}

func TestParseAgentConfigMissingFile(t *testing.T) {
	_, err := parseAgentConfig(
		[]string{"--config", filepath.Join(t.TempDir(), "missing.json")},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}
}

func TestParseAgentConfigInvalidJSON(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{invalid}`)

	_, err := parseAgentConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}
}

func TestParseAgentConfigInvalidPollInterval(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{
		"poll_interval": "invalid"
	}`)

	_, err := parseAgentConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}
}

func TestParseAgentConfigInvalidReportInterval(t *testing.T) {
	configPath := writeAgentConfigFile(t, `{
		"report_interval": "invalid"
	}`)

	_, err := parseAgentConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseAgentConfig() error = nil, want error")
	}
}

func writeAgentConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "agent.json")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write agent config file: %v", err)
	}

	return path
}
