package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	gauges, _, err := a.repo.Snapshot(context.Background())
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

	hashValue := calcHash(compressedBody, a.hashKey)

	retryDelays := []time.Duration{
		time.Second * 1,
		time.Second * 3,
		time.Second * 5,
	}

	var lastErr error

	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		req := a.client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Content-Encoding", "gzip")

		if hashValue != "" {
			req.SetHeader("HashSHA256", hashValue)
		}

		req.SetBody(compressedBody)

		resp, err := req.Post(a.baseURL + "/updates")

		/* тест:

		1. запускаем Агента

			./cmd/agent/agent -a=localhost:8080 -p=2 -r=10 -k=god

		2. запускаем netcat и ловим заголовок HashSHA256

			% nc -l 8080
				POST /updates HTTP/1.1
				Host: localhost:8080
				User-Agent: go-resty/2.17.1 (https://github.com/go-resty/resty)
				Content-Length: 391
				Accept: application/json
				Content-Encoding: gzip
				Content-Type: application/json
				Hashsha256: cacd9ec0e8aecc2dd09c9f5f37bae303b38ad0e98da5eb8a8d2b0ecd59b240ec
				Accept-Encoding: gzip

				h?ܥ???????](??a???o`??S[i?g?(ҩ?????_?yT??$2?Xb,9?????????>?U}?]?Q?p?	?Lβl?wZ)펶x??X??Oc??
																											?yK,3???(??7???<?G:C;??)<EJ%K?XX{??#w????s!"*:?+?p?jp"9?|0???7?BB?@???<P
				????w5Z?JU?Ʊ??C
							?\??
								???V?@4?D?T+???ׯ??9N??e???F?<?B?\R1l???SyL+???n=?1??????v?
				MP?)?Fח???Ǵ?7d
		*/

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

func calcHash(body []byte, key string) string {
	if key == "" {
		return ""
	}

	data := append(body, []byte(key)...)
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
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
