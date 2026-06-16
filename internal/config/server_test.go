package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseServerConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want ServerConfig
	}{
		{
			name: "defaults",
			want: ServerConfig{
				ServerAddr:         "localhost:8080",
				StoreIntervalInSec: 300,
				FileStoragePath:    "metrics-db.json",
			},
		},
		{
			name: "flags",
			args: []string{
				"-a", "server:8081",
				"-i", "60",
				"-f", "metrics.json",
				"-r",
				"-d", "postgres://flag",
				"-k", "flag-key",
				"--crypto-key", "private.pem",
				"--audit-file", "audit.log",
				"--audit-url", "http://audit",
				"-t", "192.168.1.0/24",
			},
			want: ServerConfig{
				ServerAddr:         "server:8081",
				StoreIntervalInSec: 60,
				FileStoragePath:    "metrics.json",
				Restore:            true,
				DatabaseDSN:        "postgres://flag",
				Key:                "flag-key",
				AuditFile:          "audit.log",
				AuditURL:           "http://audit",
				CryptoKey:          "private.pem",
				TrustedSubnet:      "192.168.1.0/24",
			},
		},
		{
			name: "environment overrides flags",
			args: []string{
				"-a", "flag-server:8081",
				"-i", "60",
				"-f", "flag-metrics.json",
				"-r=false",
				"-d", "postgres://flag",
				"-k", "flag-key",
				"--crypto-key", "flag-private.pem",
				"--audit-file", "flag-audit.log",
				"--audit-url", "http://flag-audit",
				"-t", "192.168.1.0/24",
			},
			env: map[string]string{
				"ADDRESS":           "env-server:8082",
				"STORE_INTERVAL":    "120",
				"FILE_STORAGE_PATH": "env-metrics.json",
				"RESTORE":           "true",
				"DATABASE_DSN":      "postgres://env",
				"KEY":               "env-key",
				"CRYPTO_KEY":        "env-private.pem",
				"AUDIT_FILE":        "env-audit.log",
				"AUDIT_URL":         "http://env-audit",
				"TRUSTED_SUBNET":    "10.0.0.0/8",
			},
			want: ServerConfig{
				ServerAddr:         "env-server:8082",
				StoreIntervalInSec: 120,
				FileStoragePath:    "env-metrics.json",
				Restore:            true,
				DatabaseDSN:        "postgres://env",
				Key:                "env-key",
				AuditFile:          "env-audit.log",
				AuditURL:           "http://env-audit",
				CryptoKey:          "env-private.pem",
				TrustedSubnet:      "10.0.0.0/8",
			},
		},
		{
			name: "store file environment alias",
			env: map[string]string{
				"FILE_STORAGE_PATH": "old-name.json",
				"STORE_FILE":        "new-name.json",
			},
			want: ServerConfig{
				ServerAddr:         "localhost:8080",
				StoreIntervalInSec: 300,
				FileStoragePath:    "new-name.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseServerConfig(tt.args, testLookupEnv(tt.env))
			if err != nil {
				t.Fatalf("parseServerConfig() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("parseServerConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseServerConfigFromFile(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"address": "json-server:8083",
		"restore": true,
		"store_interval": "45s",
		"store_file": "json-metrics.json",
		"database_dsn": "postgres://json",
		"key": "json-key",
		"crypto_key": "json-private.pem",
		"audit_file": "json-audit.log",
		"audit_url": "http://json-audit",
		"trusted_subnet": "172.16.0.0/12"
	}`)

	got, err := parseServerConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		ServerAddr:         "json-server:8083",
		StoreIntervalInSec: 45,
		FileStoragePath:    "json-metrics.json",
		Restore:            true,
		DatabaseDSN:        "postgres://json",
		Key:                "json-key",
		CryptoKey:          "json-private.pem",
		AuditFile:          "json-audit.log",
		AuditURL:           "http://json-audit",
		TrustedSubnet:      "172.16.0.0/12",
		ConfigPath:         configPath,
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigFileKeepsDefaults(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"address": "json-server:8083"
	}`)

	got, err := parseServerConfig(
		[]string{"-c", configPath},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		ServerAddr:         "json-server:8083",
		StoreIntervalInSec: 300,
		FileStoragePath:    "metrics-db.json",
		ConfigPath:         configPath,
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigFlagsOverrideFile(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"address": "json-server:8083",
		"restore": true,
		"store_interval": "45s",
		"store_file": "json-metrics.json",
		"database_dsn": "postgres://json",
		"key": "json-key",
		"crypto_key": "json-private.pem",
		"audit_file": "json-audit.log",
		"audit_url": "http://json-audit",
		"trusted_subnet": "172.16.0.0/12"
	}`)

	got, err := parseServerConfig(
		[]string{
			"--config", configPath,
			"-a", "flag-server:8084",
			"-i", "60",
			"-f", "flag-metrics.json",
			"-r=false",
			"-d", "postgres://flag",
			"-k", "flag-key",
			"--crypto-key", "flag-private.pem",
			"--audit-file", "flag-audit.log",
			"--audit-url", "http://flag-audit",
			"-t", "192.168.1.0/24",
		},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		ServerAddr:         "flag-server:8084",
		StoreIntervalInSec: 60,
		FileStoragePath:    "flag-metrics.json",
		Restore:            false,
		DatabaseDSN:        "postgres://flag",
		Key:                "flag-key",
		CryptoKey:          "flag-private.pem",
		AuditFile:          "flag-audit.log",
		AuditURL:           "http://flag-audit",
		TrustedSubnet:      "192.168.1.0/24",
		ConfigPath:         configPath,
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigEnvironmentOverridesFileAndFlags(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"address": "json-server:8083",
		"restore": false,
		"store_interval": "45s",
		"store_file": "json-metrics.json",
		"database_dsn": "postgres://json",
		"key": "json-key",
		"crypto_key": "json-private.pem",
		"audit_file": "json-audit.log",
		"audit_url": "http://json-audit",
		"trusted_subnet": "172.16.0.0/12"
	}`)

	got, err := parseServerConfig(
		[]string{
			"--config", configPath,
			"-a", "flag-server:8084",
			"-i", "60",
			"-f", "flag-metrics.json",
			"-r=false",
			"-d", "postgres://flag",
			"-k", "flag-key",
			"--crypto-key", "flag-private.pem",
			"--audit-file", "flag-audit.log",
			"--audit-url", "http://flag-audit",
			"-t", "192.168.1.0/24",
		},
		testLookupEnv(map[string]string{
			"ADDRESS":        "env-server:8085",
			"STORE_INTERVAL": "120",
			"STORE_FILE":     "env-metrics.json",
			"RESTORE":        "true",
			"DATABASE_DSN":   "postgres://env",
			"KEY":            "env-key",
			"CRYPTO_KEY":     "env-private.pem",
			"AUDIT_FILE":     "env-audit.log",
			"AUDIT_URL":      "http://env-audit",
			"TRUSTED_SUBNET": "10.0.0.0/8",
		}),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		ServerAddr:         "env-server:8085",
		StoreIntervalInSec: 120,
		FileStoragePath:    "env-metrics.json",
		Restore:            true,
		DatabaseDSN:        "postgres://env",
		Key:                "env-key",
		CryptoKey:          "env-private.pem",
		AuditFile:          "env-audit.log",
		AuditURL:           "http://env-audit",
		TrustedSubnet:      "10.0.0.0/8",
		ConfigPath:         configPath,
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigEnvironmentOverridesConfigFlag(t *testing.T) {
	flagConfigPath := writeServerConfigFile(t, `{
		"address": "flag-file-server:8081"
	}`)

	envConfigPath := writeServerConfigFile(t, `{
		"address": "env-file-server:8082"
	}`)

	got, err := parseServerConfig(
		[]string{"--config", flagConfigPath},
		testLookupEnv(map[string]string{
			"CONFIG": envConfigPath,
		}),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		ServerAddr:         "env-file-server:8082",
		StoreIntervalInSec: 300,
		FileStoragePath:    "metrics-db.json",
		ConfigPath:         envConfigPath,
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigInvalidEnvironment(t *testing.T) {
	_, err := parseServerConfig(nil, testLookupEnv(map[string]string{
		"RESTORE": "invalid",
	}))
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}

func TestParseServerConfigMissingFile(t *testing.T) {
	_, err := parseServerConfig(
		[]string{"--config", filepath.Join(t.TempDir(), "missing.json")},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}

func TestParseServerConfigInvalidJSON(t *testing.T) {
	configPath := writeServerConfigFile(t, `{invalid}`)

	_, err := parseServerConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}

func TestParseServerConfigInvalidStoreInterval(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"store_interval": "invalid"
	}`)

	_, err := parseServerConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}

func writeServerConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "server.json")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write server config file: %v", err)
	}

	return path
}
