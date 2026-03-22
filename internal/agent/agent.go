package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v4/cpu"
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
	pollSinceReport atomic.Int64
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
	if rateLimit <= 0 {
		return nil, fmt.Errorf("rate limit must be > 0, got %d", rateLimit)
	}

	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
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

// Run запускает независимые горутины:
// сбор runtime-метрик, сбор системных метрик и отправку метрик на Сервер.
func (a *Agent) Run() {
	go a.runRuntimePollLoop()
	go a.runSystemPollLoop()
	go a.runReportLoop()

	select {}
}

func (a *Agent) runRuntimePollLoop() {
	ticker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.pollRuntime()
	}
}
func (a *Agent) runSystemPollLoop() {
	ticker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	defer ticker.Stop()

	// Первый вызов нужен, чтобы инициализировать базу для cpu.Percent(0, true).
	// NOTE: https://pkg.go.dev/github.com/shirou/gopsutil/v4/cpu
	if _, err := cpu.Percent(0, true); err != nil {
		logError(fmt.Errorf("init cpu percent: %w", err))
	}

	for range ticker.C {
		a.pollSystem()
	}
}

func (a *Agent) runReportLoop() {
	ticker := time.NewTicker(time.Duration(a.reportIntervalInSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		a.report()
	}
}

func logError(err error) {
	if err == nil {
		return
	}
	log.Printf("(×﹏×) %v", err)
}
