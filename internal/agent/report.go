package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/hash"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func (a *Agent) report() {
	// запомним значение и обнулим
	pollCount := a.pollSinceReport.Swap(0)
	if pollCount == 0 {
		return
	}

	metrics, err := a.buildMetricsBatch(pollCount)
	if err != nil {
		// метод poll() в соседней горутине мог уже подинкрементить этот счетчик,
		// поэтому не восстановим, а добавим запомненное ранее значение обратно
		a.pollSinceReport.Add(pollCount)
		return
	}

	if len(metrics) == 0 {
		a.pollSinceReport.Add(pollCount)
		return
	}

	select {
	case a.sendQueue <- reportTask{
		metrics:   metrics,
		pollCount: pollCount,
	}:
	default:
		a.pollSinceReport.Add(pollCount)
		a.log.Warn("sendQueue is full")
	}
}

func (a *Agent) buildMetricsBatch(pollCount int64) ([]model.Metrics, error) {
	gauges, _, err := a.repo.Snapshot(context.Background())
	if err != nil {
		return nil, fmt.Errorf("snapshot metrics: %w", err)
	}

	// Snapshot сейчас не гарантирует одномоментную согласованность всех метрик:
	// часть gauge-метрик может быть уже обновлена другой горутиной в момент
	// формирования batch.

	metrics := make([]model.Metrics, 0, len(gauges)+1)

	for name, value := range gauges {
		v := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}

	metrics = append(metrics, model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	})

	return metrics, nil
}

func (a *Agent) sendMetrics(ctx context.Context, metrics []model.Metrics) error {
	body, err := prepareRequestBody(metrics)
	if err != nil {
		return err
	}

	hashValue := hash.CalcHash(body, a.hashKey)

	if err := a.postWithRetry(ctx, "/updates", body, hashValue); err != nil {
		return fmt.Errorf("send metrics batch: %w", err)
	}

	return nil
}

func prepareRequestBody(metrics []model.Metrics) ([]byte, error) {
	body, err := json.Marshal(metrics)
	if err != nil {
		return nil, fmt.Errorf("marshal metrics batch: %w", err)
	}

	compressedBody, err := gzipCompress(body)
	if err != nil {
		return nil, fmt.Errorf("gzip compress metrics batch: %w", err)
	}

	return compressedBody, nil
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

func isRetriableStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || (statusCode >= 500 && statusCode < 600)
}

func (a *Agent) postWithRetry(
	ctx context.Context,
	path string,
	body []byte,
	hashValue string,
) error {
	retryDelays := []time.Duration{
		time.Second * 1,
		time.Second * 3,
		time.Second * 5,
	}

	var lastErr error

	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		req := a.client.R().
			SetContext(ctx).
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip").
			SetBody(body)

		if hashValue != "" {
			req.SetHeader("HashSHA256", hashValue)
		}

		resp, err := req.Post(a.baseURL + path)
		if err == nil {
			if resp.StatusCode() == http.StatusOK {
				return nil
			}

			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode())
			if !isRetriableStatusCode(resp.StatusCode()) || attempt >= len(retryDelays) {
				return lastErr
			}

			if err := waitRetry(ctx, retryDelays[attempt]); err != nil {
				return err
			}
			continue
		}

		lastErr = err
		if !isRetriableAgentError(err) || attempt >= len(retryDelays) {
			return lastErr
		}

		if err := waitRetry(ctx, retryDelays[attempt]); err != nil {
			return err
		}
	}

	return lastErr
}

func isRetriableAgentError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
