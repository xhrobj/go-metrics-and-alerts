package handler

import (
	"fmt"
	"html"
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

// принимает метрику на хранение
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
}

// возвращает аккумулированное значение метрики в текстовом виде
func (h *Handler) Value(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	// invalid type or missing name -> 404
	if !(metricType == model.Gauge || metricType == model.Counter) || metricName == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	out := ""
	switch metricType {
	case model.Gauge:
		x, err := h.repo.GetGauge(metricName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		out = strconv.FormatFloat(x, 'f', -1, 64)
	case model.Counter:
		x, err := h.repo.GetCounter(metricName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		out = strconv.FormatInt(x, 10)
	}

	// success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(out))
}

// отдает страницу со списком имён и значений всех известных на текущий момент метрик
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	gauges, counters := h.repo.Snapshot()

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
	w.Write([]byte(out.String()))
}
