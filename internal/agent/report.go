package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func (a *Agent) report() {
	if a.pollSinceReport == 0 {
		return
	}

	metrics, err := a.buildMetricsBatch()
	if err != nil {
		return
	}

	if len(metrics) == 0 {
		return
	}

	if err := a.sendMetrics(metrics); err != nil {
		return
	}

	a.pollSinceReport = 0
}

func (a *Agent) buildMetricsBatch() ([]model.Metrics, error) {
	gauges, _, err := a.repo.Snapshot()
	if err != nil {
		return nil, fmt.Errorf("snapshot metrics: %w", err)
	}

	metrics := make([]model.Metrics, 0, len(gauges)+1)

	for name, value := range gauges {
		v := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}

	delta := int64(a.pollSinceReport)
	metrics = append(metrics, model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &delta,
	})

	return metrics, nil
}

func (a *Agent) sendMetrics(metrics []model.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics batch: %w", err)
	}

	compressedBody, err := gzipCompress(body)
	if err != nil {
		return fmt.Errorf("gzip compress metrics batch: %w", err)
	}

	retryDelays := []time.Duration{
		time.Second * 1,
		time.Second * 3,
		time.Second * 5,
	}

	var lastErr error

	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		resp, err := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(compressedBody).
			Post(a.baseURL + "/updates")

		if err == nil {
			if resp.StatusCode() != http.StatusOK {
				return fmt.Errorf("send metrics batch: unexpected status code: %d", resp.StatusCode())
			}
			return nil
		}

		lastErr = fmt.Errorf("send metrics batch: %w", err)

		if !isRetriableAgentError(err) || attempt == len(retryDelays) {
			return lastErr
		}

		time.Sleep(retryDelays[attempt])
	}

	return lastErr
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

func isRetriableAgentError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
