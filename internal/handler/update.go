package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

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
