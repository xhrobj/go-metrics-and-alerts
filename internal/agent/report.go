package agent

import (
	"fmt"
	"log"
	"net/http"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func (a *Agent) report() {
	gauges, _ := a.repo.Snapshot()

	log.Printf("%d >>> report\n", a.uptime)

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

	resp, err := a.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metric).
		Post(url)

	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status())
	}

	return nil
}

func logError(err error) {
	if err == nil {
		return
	}
	log.Printf("(×﹏×) %v", err)
}
