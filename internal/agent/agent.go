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
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

// AgentStorage описывает хранилище метрик, используемое Агентом.
type AgentStorage interface {
	UpdateGauge(context.Context, string, float64) error
	UpdateCounter(context.Context, string, int64) error
	Snapshot(context.Context) (map[string]float64, map[string]int64, error)
}

type reportTask struct {
	metrics   []model.Metrics
	pollCount int64
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

	// sendQueue - очередь задач на отправку batch-ов метрик на Сервер.
	sendQueue chan reportTask

	// pollSinceReport - количество вызовов pollRuntime() с момента последней
	// успешной отправки отчёта. Используется для формирования метрики PollCount.
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
		sendQueue:           make(chan reportTask, rateLimit),
	}, nil
}

// Run запускает независимые горутины:
// сбор runtime-метрик, сбор системных метрик и отправку метрик на Сервер.
func (a *Agent) Run() {
	go a.runRuntimePollLoop()
	go a.runSystemPollLoop()

	for i := 0; i < a.rateLimit; i++ {
		go a.runSendWorker()
	}

	a.runReportLoop()
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

	// NOTE: Первый вызов нужен, чтобы инициализировать базу для cpu.Percent(0, true).
	// https://pkg.go.dev/github.com/shirou/gopsutil/v4/cpu
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

func (a *Agent) runSendWorker() {
	for task := range a.sendQueue {
		if err := a.sendMetrics(task.metrics); err != nil {
			// Пока задача отправлялась, runtime-сборщик уже мог накопить
			// новые poll'ы, поэтому возвращаем старое значение через Add.
			a.pollSinceReport.Add(task.pollCount)
			logError(err)
		}
	}
}

func logError(err error) {
	if err == nil {
		return
	}
	log.Printf("(×﹏×) %v", err)
}
