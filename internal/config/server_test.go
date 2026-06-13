package config

import "testing"

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

func TestParseServerConfigInvalidEnvironment(t *testing.T) {
	_, err := parseServerConfig(nil, testLookupEnv(map[string]string{
		"RESTORE": "invalid",
	}))
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}
