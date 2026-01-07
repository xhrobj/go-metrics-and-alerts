package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func UpdatePage(rw http.ResponseWriter, rq *http.Request) {

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

	// missing name -> 404
	if metricName == "" {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// invalid type or value -> 400
	switch metricType {
	case model.Gauge:
		if _, err := strconv.ParseFloat(metricValue, 64); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
	case model.Counter:
		if _, err := strconv.ParseInt(metricValue, 10, 64); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
	default:
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// success
	rw.Header().Set("Content-Type", "text/plain; charset=utf-8")
	rw.WriteHeader((http.StatusOK))
	fmt.Fprintf(rw, "%s %s %s", metricType, metricName, metricValue)
}
