package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type AgentStorage interface {
	UpdateGauge(string, float64) error
	UpdateCounter(string, int64) error
	Snapshot() (map[string]float64, map[string]int64, error)
}

type Agent struct {
	repo                AgentStorage
	baseURL             string
	pollIntervalInSec   int
	reportIntervalInSec int
	client              *resty.Client
	pollSinceReport     int
}

func New(
	repo AgentStorage,
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
	pollTicker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.reportIntervalInSec) * time.Second)

	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			a.poll()
		case <-reportTicker.C:
			a.report()
		}
	}
}
