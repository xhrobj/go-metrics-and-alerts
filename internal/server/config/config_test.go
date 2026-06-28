package config

import (
	"os"
	"path/filepath"
	"strings"
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
				HTTPAddr:           "localhost:8080",
				GRPCAddr:           "localhost:50051",
				StoreIntervalInSec: 300,
				FileStoragePath:    "metrics-db.json",
			},
		},
		{
			name: "flags",
			args: []string{
				"-a", "server:8081",
				"-g", "grpc-server:3201",
				"--grpc-tls-cert", "flag-cert.pem",
				"--grpc-tls-key", "flag-tls-key.pem",
				"-i", "60",
				"-f", "metrics.json",
				"-r",
				"-d", "postgres://flag",
				"-k", "flag-key",
				"--crypto-key", "private.pem",
				"-t", "192.168.1.0/24",
				"--audit-file", "audit.log",
				"--audit-url", "http://audit",
			},
			want: ServerConfig{
				HTTPAddr:           "server:8081",
				GRPCAddr:           "grpc-server:3201",
				GRPCTLSCert:        "flag-cert.pem",
				GRPCTLSKey:         "flag-tls-key.pem",
				StoreIntervalInSec: 60,
				FileStoragePath:    "metrics.json",
				Restore:            true,
				DatabaseDSN:        "postgres://flag",
				Key:                "flag-key",
				CryptoKey:          "private.pem",
				TrustedSubnet:      "192.168.1.0/24",
				AuditFile:          "audit.log",
				AuditURL:           "http://audit",
			},
		},
		{
			name: "environment overrides flags",
			args: []string{
				"-a", "flag-server:8081",
				"-g", "flag-grpc:3201",
				"--grpc-tls-cert", "flag-cert.pem",
				"--grpc-tls-key", "flag-tls-key.pem",
				"-i", "60",
				"-f", "flag-metrics.json",
				"-r=false",
				"-d", "postgres://flag",
				"-k", "flag-key",
				"--crypto-key", "flag-private.pem",
				"-t", "192.168.1.0/24",
				"--audit-file", "flag-audit.log",
				"--audit-url", "http://flag-audit",
			},
			env: map[string]string{
				"ADDRESS":           "env-server:8082",
				"GRPC_ADDRESS":      "env-grpc:3202",
				"GRPC_TLS_CERT":     "env-cert.pem",
				"GRPC_TLS_KEY":      "env-tls-key.pem",
				"STORE_INTERVAL":    "120",
				"FILE_STORAGE_PATH": "env-metrics.json",
				"RESTORE":           "true",
				"DATABASE_DSN":      "postgres://env",
				"KEY":               "env-key",
				"CRYPTO_KEY":        "env-private.pem",
				"TRUSTED_SUBNET":    "10.0.0.0/8",
				"AUDIT_FILE":        "env-audit.log",
				"AUDIT_URL":         "http://env-audit",
			},
			want: ServerConfig{
				HTTPAddr:           "env-server:8082",
				GRPCAddr:           "env-grpc:3202",
				GRPCTLSCert:        "env-cert.pem",
				GRPCTLSKey:         "env-tls-key.pem",
				StoreIntervalInSec: 120,
				FileStoragePath:    "env-metrics.json",
				Restore:            true,
				DatabaseDSN:        "postgres://env",
				Key:                "env-key",
				CryptoKey:          "env-private.pem",
				TrustedSubnet:      "10.0.0.0/8",
				AuditFile:          "env-audit.log",
				AuditURL:           "http://env-audit",
			},
		},
		{
			name: "store file environment alias",
			env: map[string]string{
				"FILE_STORAGE_PATH": "old-name.json",
				"STORE_FILE":        "new-name.json",
			},
			want: ServerConfig{
				HTTPAddr:           "localhost:8080",
				GRPCAddr:           "localhost:50051",
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
		"grpc_address": "json-grpc:3203",
		"grpc_tls_cert": "json-cert.pem",
		"grpc_tls_key": "json-tls-key.pem",
		"store_interval": "45s",
		"store_file": "json-metrics.json",
		"restore": true,
		"database_dsn": "postgres://json",
		"key": "json-key",
		"crypto_key": "json-private.pem",
		"trusted_subnet": "172.16.0.0/12",
		"audit_file": "json-audit.log",
		"audit_url": "http://json-audit"
	}`)

	got, err := parseServerConfig(
		[]string{"--config", configPath},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		HTTPAddr:           "json-server:8083",
		GRPCAddr:           "json-grpc:3203",
		GRPCTLSCert:        "json-cert.pem",
		GRPCTLSKey:         "json-tls-key.pem",
		StoreIntervalInSec: 45,
		FileStoragePath:    "json-metrics.json",
		Restore:            true,
		DatabaseDSN:        "postgres://json",
		Key:                "json-key",
		CryptoKey:          "json-private.pem",
		TrustedSubnet:      "172.16.0.0/12",
		AuditFile:          "json-audit.log",
		AuditURL:           "http://json-audit",
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
		HTTPAddr:           "json-server:8083",
		GRPCAddr:           "localhost:50051",
		StoreIntervalInSec: 300,
		FileStoragePath:    "metrics-db.json",
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigFlagsOverrideFile(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"address": "json-server:8083",
		"grpc_address": "json-grpc:3203",
		"grpc_tls_cert": "json-cert.pem",
		"grpc_tls_key": "json-tls-key.pem",
		"store_interval": "45s",
		"store_file": "json-metrics.json",
		"restore": true,
		"database_dsn": "postgres://json",
		"key": "json-key",
		"crypto_key": "json-private.pem",
		"trusted_subnet": "172.16.0.0/12",
		"audit_file": "json-audit.log",
		"audit_url": "http://json-audit"
	}`)

	got, err := parseServerConfig(
		[]string{
			"-a", "flag-server:8084",
			"-g", "flag-grpc:3204",
			"--grpc-tls-cert", "flag-cert.pem",
			"--grpc-tls-key", "flag-tls-key.pem",
			"-i", "60",
			"-f", "flag-metrics.json",
			"-r=false",
			"-d", "postgres://flag",
			"-k", "flag-key",
			"--crypto-key", "flag-private.pem",
			"-t", "192.168.1.0/24",
			"--audit-file", "flag-audit.log",
			"--audit-url", "http://flag-audit",
			"--config", configPath,
		},
		testLookupEnv(nil),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		HTTPAddr:           "flag-server:8084",
		GRPCAddr:           "flag-grpc:3204",
		GRPCTLSCert:        "flag-cert.pem",
		GRPCTLSKey:         "flag-tls-key.pem",
		StoreIntervalInSec: 60,
		FileStoragePath:    "flag-metrics.json",
		Restore:            false,
		DatabaseDSN:        "postgres://flag",
		Key:                "flag-key",
		CryptoKey:          "flag-private.pem",
		TrustedSubnet:      "192.168.1.0/24",
		AuditFile:          "flag-audit.log",
		AuditURL:           "http://flag-audit",
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigEnvironmentOverridesFileAndFlags(t *testing.T) {
	configPath := writeServerConfigFile(t, `{
		"address": "json-server:8083",
		"grpc_address": "json-grpc:3203",
		"grpc_tls_cert": "json-cert.pem",
		"grpc_tls_key": "json-tls-key.pem",
		"store_interval": "45s",
		"store_file": "json-metrics.json",
		"restore": false,
		"database_dsn": "postgres://json",
		"key": "json-key",
		"crypto_key": "json-private.pem",
		"trusted_subnet": "172.16.0.0/12",
		"audit_file": "json-audit.log",
		"audit_url": "http://json-audit"
	}`)

	got, err := parseServerConfig(
		[]string{
			"-a", "flag-server:8084",
			"-g", "flag-grpc:3204",
			"--grpc-tls-cert", "flag-cert.pem",
			"--grpc-tls-key", "flag-tls-key.pem",
			"-i", "60",
			"-f", "flag-metrics.json",
			"-r=false",
			"-d", "postgres://flag",
			"-k", "flag-key",
			"--crypto-key", "flag-private.pem",
			"-t", "192.168.1.0/24",
			"--audit-file", "flag-audit.log",
			"--audit-url", "http://flag-audit",
			"--config", configPath,
		},
		testLookupEnv(map[string]string{
			"ADDRESS":        "env-server:8085",
			"GRPC_ADDRESS":   "env-grpc:3205",
			"GRPC_TLS_CERT":  "env-cert.pem",
			"GRPC_TLS_KEY":   "env-tls-key.pem",
			"STORE_INTERVAL": "120",
			"STORE_FILE":     "env-metrics.json",
			"RESTORE":        "true",
			"DATABASE_DSN":   "postgres://env",
			"KEY":            "env-key",
			"CRYPTO_KEY":     "env-private.pem",
			"TRUSTED_SUBNET": "10.0.0.0/8",
			"AUDIT_FILE":     "env-audit.log",
			"AUDIT_URL":      "http://env-audit",
		}),
	)
	if err != nil {
		t.Fatalf("parseServerConfig() error = %v", err)
	}

	want := ServerConfig{
		HTTPAddr:           "env-server:8085",
		GRPCAddr:           "env-grpc:3205",
		GRPCTLSCert:        "env-cert.pem",
		GRPCTLSKey:         "env-tls-key.pem",
		StoreIntervalInSec: 120,
		FileStoragePath:    "env-metrics.json",
		Restore:            true,
		DatabaseDSN:        "postgres://env",
		Key:                "env-key",
		CryptoKey:          "env-private.pem",
		TrustedSubnet:      "10.0.0.0/8",
		AuditFile:          "env-audit.log",
		AuditURL:           "http://env-audit",
	}

	if got != want {
		t.Fatalf("parseServerConfig() = %+v, want %+v", got, want)
	}
}

func TestParseServerConfigEnvironmentOverridesConfigFlag(t *testing.T) {
	flagConfigPath := writeServerConfigFile(t, `{
		"address": "flag-file-server:8081",
		"grpc_address": "flag-file-grpc:3201"
	}`)

	envConfigPath := writeServerConfigFile(t, `{
		"address": "env-file-server:8082",
		"grpc_address": "env-file-grpc:3202"
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
		HTTPAddr:           "env-file-server:8082",
		GRPCAddr:           "env-file-grpc:3202",
		StoreIntervalInSec: 300,
		FileStoragePath:    "metrics-db.json",
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

	wantError := `RESTORE="invalid"`
	if !strings.Contains(err.Error(), wantError) {
		t.Fatalf("parseServerConfig() error = %q, want substring %q", err, wantError)
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
