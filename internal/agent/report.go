package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func (a *Agent) report() {
	gauges, _ := a.repo.Snapshot()

	log.Printf(">>> report\n")

	for name, value := range gauges {
		err := a.sendGauge(name, value)
		if err != nil {
			logError(err)
		}
	}

	err := a.sendCounter("PollCount", int64(a.pollSinceReport))
	if err != nil {
		logError(err)
		return
	}

	a.pollSinceReport = 0
}

func (a *Agent) sendGauge(name string, value float64) error {
	metric := model.Metrics{
		ID:    name,
		MType: model.Gauge,
		Value: &value,
	}
	return a.sendMetric(metric)
}

func (a *Agent) sendCounter(name string, delta int64) error {
	metric := model.Metrics{
		ID:    name,
		MType: model.Counter,
		Delta: &delta,
	}
	return a.sendMetric(metric)
}

func (a *Agent) sendMetric(metric model.Metrics) error {
	url := a.baseURL + "/update"
	log.Printf("* %s", url)

	body, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	compressedBody, err := gzipCompress(body)
	if err != nil {
		return fmt.Errorf("failed to compress metric: %w", err)
	}

	resp, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compressedBody).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status())
	}

	return nil
}

func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	zw := gzip.NewWriter(&buf)

	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func logError(err error) {
	if err == nil {
		return
	}
	log.Printf("(×﹏×) %v", err)
}
