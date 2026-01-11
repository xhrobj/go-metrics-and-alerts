package agent

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func (a *Agent) report() {
	gauges, counters := a.repo.Snapshot()

	fmt.Printf("%d >>> report\n", a.uptime)

	for name, value := range gauges {
		err := a.sendGauge(name, value)
		if err != nil {
			logError(err)
		}
	}

	for name, value := range counters {
		err := a.sendCounter(name, value)
		if err != nil {
			logError(err)
		}
	}
}

func (a *Agent) sendGauge(name string, value float64) error {
	path := model.Gauge + "/" + name + "/" + strconv.FormatFloat(value, 'f', -1, 64)
	return a.sendMetric(path)
}

func (a *Agent) sendCounter(name string, value int64) error {
	path := model.Counter + "/" + name + "/" + strconv.FormatInt(value, 10)
	return a.sendMetric(path)
}

func (a *Agent) sendMetric(path string) error {
	url := a.baseURL + "/update/" + path
	fmt.Println(" *", url)

	resp, err := a.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status())
	}

	return nil
}

func logError(err error) {
	fmt.Println("\t(×﹏×)", err)
}
