package agent

import (
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

const pollIntervalInSec = 2
const reportIntervalInSec = 10

type Agent struct {
	repo    repository.AgentStorage
	baseURL string
	uptime  int
}

func New(repo repository.AgentStorage, baseURL string) *Agent {
	return &Agent{repo: repo, baseURL: baseURL}
}

func (a *Agent) Run() {
	for {
		a.tick()
		time.Sleep(1 * time.Second)

	}
}

func (a *Agent) tick() {
	if a.uptime%pollIntervalInSec == 0 {
		a.poll()
	}
	if a.uptime != 0 && a.uptime%reportIntervalInSec == 0 {
		a.report()
	}
	a.uptime++
}
