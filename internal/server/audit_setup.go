package server

import (
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/server/config"
	"go.uber.org/zap"
)

func setupAudit(
	cfg config.ServerConfig,
	log *zap.Logger,
) (*audit.Auditor, func(), error) {
	noop := func() {
		// 4Sonar: Аудит может быть отключен, освобождать ресурсы не требуется
	}

	if cfg.AuditFile == "" && cfg.AuditURL == "" {
		return nil, noop, nil
	}

	cleanup := noop
	auditor := audit.NewAuditor()

	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, nil, err
		}

		cleanup = func() {
			if err := fileObserver.Close(); err != nil {
				log.Error("failed to close audit file", zap.Error(err))
			}
		}

		auditor.Subscribe(fileObserver)
	}

	if cfg.AuditURL != "" {
		auditor.Subscribe(audit.NewRemoteObserver(cfg.AuditURL))
	}

	return auditor, cleanup, nil
}
