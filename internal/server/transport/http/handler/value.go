package handler

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
)

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
	w.Header().Set(protocol.HeaderContentType, protocol.ContentTypeTextPlainUTF8)
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
	w.Header().Set(protocol.HeaderContentType, protocol.ContentTypeHTMLUTF8)
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
