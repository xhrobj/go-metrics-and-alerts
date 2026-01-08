package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

type Handler struct {
	repo repository.ServerStorage
}

func New(repo repository.ServerStorage) *Handler {
	return &Handler{repo}
}

func (h *Handler) UpdatePage(rw http.ResponseWriter, rq *http.Request) {
	// method must be POST
	if rq.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// invalid Content-Type
	ct := rq.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(strings.ToLower(ct), "text/plain") {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// expect: /update/{type}/{name}/{value}
	path := strings.TrimPrefix(rq.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 4 {
		// malformed path -> 404
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	metricType, metricName, metricValue := parts[1], parts[2], parts[3]

	// invalid type or value -> 400
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		h.repo.UpdateGauge(metricName, value)
	case model.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(metricName, value)
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// success
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader((http.StatusOK))

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
