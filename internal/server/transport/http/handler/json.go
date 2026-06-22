package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/protocol"
)

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, dst any) bool {
	// method must be POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return false
	}

	// invalid Content-Type
	ct := r.Header.Get(protocol.HeaderContentType)
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), protocol.ContentTypeJSON) {
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
	w.Header().Set(protocol.HeaderContentType, protocol.ContentTypeJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(metric)
}
