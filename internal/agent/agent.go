package agent

import (
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

type Agent struct {
	repo   *repository.MemStorage
	uptime int
}

func New(repo *repository.MemStorage) *Agent {
	return &Agent{repo: repo}
}

func (a *Agent) Run() {
	for {
		if a.uptime%2 == 0 {
			a.poll()
		}
		if a.uptime%10 == 0 {
			a.report()
		}
		time.Sleep(1 * time.Second)
		a.uptime++
	}
}
