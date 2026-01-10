package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

type Handler struct {
	repo repository.ServerStorage
}

func New(repo repository.ServerStorage) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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
		h.repo.UpdateGauge(metricName, value)
	case model.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(metricName, value)
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	fmt.Printf("-> %s %s %s\n", metricType, metricName, metricValue)
	switch metricType {
	case model.Gauge:
		x, _ := h.repo.GetGauge(metricName)
		fmt.Printf("\t%s\n", strconv.FormatFloat(x, 'f', -1, 64))
	case model.Counter:
		x, _ := h.repo.GetCounter(metricName)
		fmt.Printf("\t%d\n", x)
	}
}
