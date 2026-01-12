package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

type Agent struct {
	repo                repository.AgentStorage
	baseURL             string
	pollIntervalInSec   int
	reportIntervalInSec int
	client              *resty.Client
	uptime              int
}

func New(
	repo repository.AgentStorage,
	baseURL string,
	pollIntervalInSec int,
	reportIntervalInSec int,
) (*Agent, error) {
	if pollIntervalInSec <= 0 {
		return nil, fmt.Errorf("poll interval must be > 0, got %d", pollIntervalInSec)
	}
	if reportIntervalInSec <= 0 {
		return nil, fmt.Errorf("report interval must be > 0, got %d", reportIntervalInSec)
	}

	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	return &Agent{
		repo:                repo,
		baseURL:             baseURL,
		pollIntervalInSec:   pollIntervalInSec,
		reportIntervalInSec: reportIntervalInSec,
		client:              resty.New(),
	}, nil
}

func (a *Agent) Run() {
	for {
		a.tick()
		time.Sleep(1 * time.Second)
	}
}

func (a *Agent) tick() {
	if a.uptime%a.pollIntervalInSec == 0 {
		a.poll()
	}
	if a.uptime != 0 && a.uptime%a.reportIntervalInSec == 0 {
		a.report()
	}
	a.uptime++
}
