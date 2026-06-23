package server

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"go.uber.org/zap"
)

func TestSetupAuditDisabled(t *testing.T) {
	auditor, cleanup, err := setupAudit(config.ServerConfig{}, zap.NewNop())

	require.NoError(t, err)
	require.Nil(t, auditor)
	require.NotNil(t, cleanup)
	require.NotPanics(t, cleanup)
}

func TestSetupAuditReturnsReusableAuditor(t *testing.T) {
	auditFile := filepath.Join(t.TempDir(), "audit.log")

	auditor, cleanup, err := setupAudit(config.ServerConfig{
		AuditFile: auditFile,
	}, zap.NewNop())
	require.NoError(t, err)
	require.NotNil(t, auditor)
	require.NotNil(t, cleanup)
	t.Cleanup(cleanup)

	event := audit.Event{
		TS:        42,
		Metrics:   []string{"Alloc", "PollCount"},
		IPAddress: "203.0.113.10",
	}

	require.NoError(t, auditor.Notify(context.Background(), event))
	cleanup()

	data, err := os.ReadFile(auditFile)
	require.NoError(t, err)

	var got audit.Event
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, event, got)
}

func TestSetupAuditReturnsErrorForInvalidFile(t *testing.T) {
	auditFile := filepath.Join(t.TempDir(), "missing", "audit.log")

	auditor, cleanup, err := setupAudit(config.ServerConfig{
		AuditFile: auditFile,
	}, zap.NewNop())

	require.ErrorContains(t, err, "open audit file")
	require.Nil(t, auditor)
	require.Nil(t, cleanup)
}
