package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// AgentStorage описывает хранилище метрик, используемое Агентом.
type AgentStorage interface {
	UpdateGauge(context.Context, string, float64) error
	UpdateCounter(context.Context, string, int64) error
	Snapshot(context.Context) (map[string]float64, map[string]int64, error)
}

// Agent собирает runtime-метрики и отправляет их на сервер по HTTP.
type Agent struct {
	repo                AgentStorage
	baseURL             string
	pollIntervalInSec   int
	reportIntervalInSec int
	rateLimit           int
	hashKey             string
	client              *resty.Client

	// pollSinceReport — количество вызовов poll() с момента последней отправки
	// отчёта. Используется для формирования метрики PollCount.
	pollSinceReport int
}

// New создаёт нового Агента с указанными параметрами конфигурации.
func New(
	repo AgentStorage,
	baseURL string,
	pollIntervalInSec int,
	reportIntervalInSec int,
	rateLimit int,
	hashKey string,
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

	if rateLimit <= 0 {
		return nil, fmt.Errorf("rate limit must be > 0, got %d", rateLimit)
	}

	return &Agent{
		repo:                repo,
		baseURL:             baseURL,
		pollIntervalInSec:   pollIntervalInSec,
		reportIntervalInSec: reportIntervalInSec,
		rateLimit:           rateLimit,
		hashKey:             hashKey,
		client:              resty.New(),
	}, nil
}

// Run запускает цикл работы агента: периодический сбор и отправку метрик.
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
