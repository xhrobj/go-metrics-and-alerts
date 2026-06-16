package handler

import (
	"context"
	"database/sql"

	"github.com/xhrobj/go-metrics-and-alerts/internal/audit"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"go.uber.org/zap"
)

// Service описывает бизнес-логику работы с метриками,
// используемую HTTP-обработчиками.
type Service interface {
	UpdateGauge(context.Context, string, float64) error
	UpdateCounter(context.Context, string, int64) (int64, error)

	UpdateMetrics(context.Context, []model.Metrics) error

	GetGauge(context.Context, string) (float64, error)
	GetCounter(context.Context, string) (int64, error)

	Snapshot(context.Context) (map[string]float64, map[string]int64, error)
}

// Auditor описывает диспетчер событий аудита.
type Auditor interface {
	Notify(context.Context, audit.Event) error
}

// Handler обрабатывает HTTP-запросы, связанные с метриками.
type Handler struct {
	service Service
	db      *sql.DB
	auditor Auditor
	log     *zap.Logger
}

// New создаёт новый Handler, использующий переданный сервис метрик и соединение с БД
func New(service Service, db *sql.DB) *Handler {
	return &Handler{
		service: service,
		db:      db,
	}
}

// EnableAudit подключает аудит успешной обработки метрик.
func (h *Handler) EnableAudit(auditor Auditor, log *zap.Logger) {
	h.auditor = auditor
	h.log = log
}
