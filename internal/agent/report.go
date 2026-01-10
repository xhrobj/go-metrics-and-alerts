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

	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")

	response, err := a.client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", response.Status)
	}

	return nil
}

func logError(err error) {
	fmt.Println("  (×﹏×)", err)
}
