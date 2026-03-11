package handler

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

type ServerStorage interface {
	UpdateGauge(string, float64)
	UpdateCounter(string, int64)
	Snapshot() (map[string]float64, map[string]int64)
	GetGauge(string) (float64, error)
	GetCounter(string) (int64, error)
}

type Handler struct {
	repo            ServerStorage
	syncFileStorage *repository.FileStorage
}

func New(repo ServerStorage) *Handler {
	return &Handler{repo: repo}
}

// SetSyncPersistence включает синхронное сохранение метрик.
func (h *Handler) SetSyncPersistence(fileStorage *repository.FileStorage) {
	h.syncFileStorage = fileStorage
}

// Update принимает метрику на хранение (данные метрики передаются через параметры URL)
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

		if err := h.saveIfSync(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case model.Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		h.repo.UpdateCounter(metricName, value)

		if err := h.saveIfSync(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

// UpdateJSON принимает метрику на хранение (данные передаются в теле POST-запроса)
func (h *Handler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	// method must be POST
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// invalid Content-Type
	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var metric model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// missing name -> 404
	if metric.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// invalid type or value -> 400
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.repo.UpdateGauge(metric.ID, *metric.Value)

		if err := h.saveIfSync(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Gauge,
			Value: metric.Value,
		})

	case model.Counter:
		if metric.Delta == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		h.repo.UpdateCounter(metric.ID, *metric.Delta)

		if err := h.saveIfSync(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		total, err := h.repo.GetCounter(metric.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Counter,
			Delta: &total,
		})

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

// Value возвращает текущее значение метрики в текстовом виде.
// Данные о метрике передаются через параметры URL
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
		value, err := h.repo.GetGauge(metricName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		out = strconv.FormatFloat(value, 'f', -1, 64)
	case model.Counter:
		value, err := h.repo.GetCounter(metricName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		out = strconv.FormatInt(value, 10)
	}

	// success
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(out))
}

// ValueJSON возвращает текущее значение метрики в формате JSON.
// Идентификатор и тип метрики передаются в теле POST-запроса
func (h *Handler) ValueJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ct := r.Header.Get("Content-Type")
	if ct == "" || !strings.HasPrefix(strings.ToLower(ct), "application/json") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var metric model.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// missing name -> 404
	if metric.ID == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch metric.MType {
	case model.Gauge:
		value, err := h.repo.GetGauge(metric.ID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, model.Metrics{
				ID:    metric.ID,
				MType: model.Gauge,
			})
			return
		}

		writeJSON(w, http.StatusOK, model.Metrics{
			ID:    metric.ID,
			MType: model.Gauge,
			Value: &value,
		})

	case model.Counter:
		value, err := h.repo.GetCounter(metric.ID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, model.Metrics{
				ID:    metric.ID,
				MType: model.Counter,
			})
			return
		}

		writeJSON(w, http.StatusOK, model.Metrics{
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

func writeJSON(w http.ResponseWriter, status int, metric model.Metrics) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(metric)
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

func (h *Handler) saveIfSync() error {
	if h.syncFileStorage == nil {
		return nil
	}

	return h.syncFileStorage.Save(h.repo)
}
