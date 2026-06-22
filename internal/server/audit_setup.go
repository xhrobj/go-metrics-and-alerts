package server

import (
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/transport/http/handler"
	"go.uber.org/zap"
)

func setupAudit(
	cfg config.ServerConfig,
	h *handler.Handler,
	log *zap.Logger,
) (func(), error) {
	noop := func() {
		// Освобождать ресурсы не требуется.
	}

	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return noop, nil
	}

	cleanup := noop
	auditDispatcher := audit.NewAuditor()

	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, err
		}

		cleanup = func() {
			if err := fileObserver.Close(); err != nil {
				log.Error("failed to close audit file", zap.Error(err))
			}
		}

		auditDispatcher.Subscribe(fileObserver)
	}

	if cfg.AuditURL != "" {
		auditDispatcher.Subscribe(audit.NewRemoteObserver(cfg.AuditURL))
	}

	h.EnableAudit(auditDispatcher, log)

	return cleanup, nil
}
