package config

import (
	"strings"
	"testing"
)

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

func TestParseServerConfigValidatesResult(t *testing.T) {
	_, err := parseServerConfig(
		[]string{"-t", "invalid"},
		testLookupEnv(nil),
	)
	if err == nil {
		t.Fatal("parseServerConfig() error = nil, want error")
	}
}
