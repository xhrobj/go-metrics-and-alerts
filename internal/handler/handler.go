package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
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

// Update принимает метрику на хранение.
// Данные метрики передаются через параметры URL.
//
// Legacy endpoint: для новых интеграций используйте UpdateJSON.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// method must be POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// invalid Content-Type
	ct := r.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(strings.ToLower(ct), "text/plain") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	// missing name -> 404
	if metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// invalid type or value -> 400
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := h.service.UpdateGauge(ctx, metricName, value); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case model.Counter:
		delta, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if _, err := h.service.UpdateCounter(ctx, metricName, delta); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.notifyAuditMetric(r, metricName)

	// success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// UpdateJSON принимает метрику на хранение.
// Данные метрики передаются в теле POST-запроса в формате JSON.
func (h *Handler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var metric model.Metrics
	if !decodeJSONRequest(w, r, &metric) {
		return
	}

	status := validateMetric(metric)
	if status != http.StatusOK {
		w.WriteHeader(status)
		return
	}

	var response model.Metrics

	switch metric.MType {
	case model.Gauge:
		if err := h.service.UpdateGauge(ctx, metric.ID, *metric.Value); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response = model.Metrics{
			ID:    metric.ID,
			MType: model.Gauge,
			Value: metric.Value,
		}

	case model.Counter:
		total, err := h.service.UpdateCounter(ctx, metric.ID, *metric.Delta)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		response = model.Metrics{
			ID:    metric.ID,
			MType: model.Counter,
			Delta: &total,
		}

	default:
		// NOTE: защитная ветка - validateMetric уже должен был отфильтровать некорректный тип
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.notifyAuditMetric(r, metric.ID)
	writeMetricJSON(w, http.StatusOK, response)
}

// UpdatesJSON принимает набор метрик на хранение.
// Данные метрик передаются в теле POST-запроса в формате JSON.
func (h *Handler) UpdatesJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var metrics []model.Metrics
	if !decodeJSONRequest(w, r, &metrics) {
		return
	}

	for _, metric := range metrics {
		status := validateMetric(metric)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
	}

	if err := h.service.UpdateMetrics(ctx, metrics); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.notifyAuditMetrics(r, metrics)

	w.WriteHeader(http.StatusOK)
}

// Value возвращает текущее значение метрики в текстовом виде.
// Тип и имя метрики передаются через параметры URL.
//
// Legacy endpoint: для новых интеграций используйте ValueJSON.
func (h *Handler) Value(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	// invalid type or missing name -> 404
	if (metricType != model.Gauge && metricType != model.Counter) || metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	out := ""
	switch metricType {
	case model.Gauge:
		value, err := h.service.GetGauge(ctx, metricName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		out = strconv.FormatFloat(value, 'f', -1, 64)
	case model.Counter:
		value, err := h.service.GetCounter(ctx, metricName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		out = strconv.FormatInt(value, 10)
	}

	// success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(out)); err != nil {
		return
	}
}

// ValueJSON возвращает текущее значение метрики в формате JSON.
// Идентификатор и тип метрики передаются в теле POST-запроса.
func (h *Handler) ValueJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var metric model.Metrics
	if !decodeJSONRequest(w, r, &metric) {
		return
	}

	// missing name -> 404
	if metric.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch metric.MType {
	case model.Gauge:
		value, err := h.service.GetGauge(ctx, metric.ID)
		if err != nil {
			writeMetricJSON(w, http.StatusNotFound, model.Metrics{
				ID:    metric.ID,
				MType: model.Gauge,
			})
			return
		}

		writeMetricJSON(w, http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Gauge,
			Value: &value,
		})

	case model.Counter:
		value, err := h.service.GetCounter(ctx, metric.ID)
		if err != nil {
			writeMetricJSON(w, http.StatusNotFound, model.Metrics{
				ID:    metric.ID,
				MType: model.Counter,
			})
			return
		}

		writeMetricJSON(w, http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Counter,
			Delta: &value,
		})

	default:
		// invalid type -> 404
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

// Index возвращает HTML-страницу со списком всех известных метрик
// и их текущих значений.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	gauges, counters, err := h.service.Snapshot(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var out strings.Builder
	out.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	out.WriteString("<title>storage values</title></head><body><ul>")
	for n, v := range gauges {
		safeName := html.EscapeString(n)
		value := strconv.FormatFloat(v, 'f', -1, 64)
		fmt.Fprintf(&out, "<li>%s: <strong>%s</strong></li>", safeName, value)
	}

	for n, v := range counters {
		safeName := html.EscapeString(n)
		fmt.Fprintf(&out, "<li>%s: <strong>%d</strong></li>", safeName, v)
	}
	out.WriteString("</ul></body></html>")

	// success
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(out.String())); err != nil {
		return
	}
}

// Ping проверяет соединение с базой данных.
func (h *Handler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := h.db.PingContext(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	// method must be POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	// invalid Content-Type
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return false
	}

	return true
}

func validateMetric(metric model.Metrics) int {
	// missing name -> 404
	// NOTE: Требование из С1И1:
	// "При попытке передать запрос без имени метрики возвращать http.StatusNotFound"
	if metric.ID == "" {
		return http.StatusNotFound
	}

	// invalid type or value -> 400
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return http.StatusBadRequest
		}
	case model.Counter:
		if metric.Delta == nil {
			return http.StatusBadRequest
		}
	default:
		return http.StatusBadRequest
	}

	return http.StatusOK
}

func writeMetricJSON(w http.ResponseWriter, status int, metric model.Metrics) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(metric)
}

func (h *Handler) notifyAuditMetric(r *http.Request, metricName string) {
	if h.auditor == nil || metricName == "" {
		return
	}

	h.notifyAudit(r, []string{metricName})
}

func (h *Handler) notifyAuditMetrics(r *http.Request, metrics []model.Metrics) {
	if h.auditor == nil || len(metrics) == 0 {
		return
	}

	h.notifyAudit(r, metricIDs(metrics))
}

func (h *Handler) notifyAudit(r *http.Request, metrics []string) {
	if h.auditor == nil || len(metrics) == 0 {
		return
	}

	event := audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: clientIP(r),
	}

	if err := h.auditor.Notify(r.Context(), event); err != nil && h.log != nil {
		h.log.Error("audit failed", zap.Error(err))
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func metricIDs(metrics []model.Metrics) []string {
	ids := make([]string, 0, len(metrics))

	for _, metric := range metrics {
		if metric.ID != "" {
			ids = append(ids, metric.ID)
		}
	}

	return ids
}
