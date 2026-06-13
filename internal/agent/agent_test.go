package agent

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
	"go.uber.org/zap"
)

// TestNewReturnsErrorForInvalidCryptoKey проверяет, что Агент
// не создаётся, если публичный ключ не удалось загрузить.
func TestNewReturnsErrorForInvalidCryptoKey(t *testing.T) {
	cfg := config.AgentConfig{
		ServerAddr:          "localhost:8080",
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
		CryptoKey:           filepath.Join(t.TempDir(), "missing-public.pem"),
	}

	_, err := New(repository.NewMemStorage(), cfg, zap.NewNop())
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}

	if got, want := err.Error(), "load public key"; !strings.Contains(got, want) {
		t.Fatalf("New() error = %q, want error containing %q", got, want)
	}
}
