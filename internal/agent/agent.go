package agent

import (
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
	reportIntervalInSec int) *Agent {
	return &Agent{
		repo:                repo,
		baseURL:             baseURL,
		pollIntervalInSec:   pollIntervalInSec,
		reportIntervalInSec: reportIntervalInSec,
		client:              resty.New(),
	}
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
