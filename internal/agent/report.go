package agent

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

const host = "http://localhost:8080"

var client = &http.Client{}

func (a *Agent) report() {
	gauges, counters := a.repo.Snapshot()

	fmt.Printf("%d >>> report\n", a.uptime)

	for name, value := range gauges {
		err := sendGauge(name, value)
		if err != nil {
			print(err)
		}
	}

	for name, value := range counters {
		err := sendCounter(name, value)
		if err != nil {
			print(err)
		}
	}
}

func sendGauge(name string, value float64) error {
	path := model.Gauge + "/" + name + "/" + strconv.FormatFloat(value, 'f', -1, 64)
	return sendMetric(path)

}

func sendCounter(name string, value int64) error {
	path := model.Counter + "/" + name + "/" + strconv.FormatInt(value, 10)
	return sendMetric(path)
}

func sendMetric(path string) error {
	url := host + "/update/" + path
	fmt.Println(" *", url)

	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", response.Status)
	}

	return nil
}

func print(err error) {
	fmt.Println("  (×﹏×)", err)
}
