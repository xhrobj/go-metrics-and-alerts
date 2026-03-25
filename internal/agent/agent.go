package agent

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v4/cpu"
	agentConfig "github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"go.uber.org/zap"
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

// Agent собирает runtime- и системные метрики и отправляет их на Сервер по HTTP.
type Agent struct {
	repo                AgentStorage
	baseURL             string
	pollIntervalInSec   int
	reportIntervalInSec int
	rateLimit           int
	hashKey             string
	client              *resty.Client
	log                 *zap.Logger

	// sendQueue хранит задачи на отправку batch-ов метрик на Сервер;
	// запись в очередь выполняет report(), чтение - worker'ы.
	sendQueue chan reportTask

	// pollSinceReport - количество вызовов pollRuntime() с момента последней
	// успешной отправки отчёта. Используется для формирования метрики PollCount.
	pollSinceReport atomic.Int64
}

// New создаёт нового Агента с указанными параметрами конфигурации.
func New(repo AgentStorage, cfg agentConfig.Config, log *zap.Logger) (*Agent, error) {
	if log == nil {
		log = zap.NewNop()
	}

	if cfg.PollIntervalInSec <= 0 {
		return nil, fmt.Errorf("poll interval must be > 0, got %d", cfg.PollIntervalInSec)
	}
	if cfg.ReportIntervalInSec <= 0 {
		return nil, fmt.Errorf("report interval must be > 0, got %d", cfg.ReportIntervalInSec)
	}
	if cfg.RateLimit <= 0 {
		return nil, fmt.Errorf("rate limit must be > 0, got %d", cfg.RateLimit)
	}

	baseURL := cfg.ServerAddr
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	return &Agent{
		repo:                repo,
		baseURL:             baseURL,
		pollIntervalInSec:   cfg.PollIntervalInSec,
		reportIntervalInSec: cfg.ReportIntervalInSec,
		rateLimit:           cfg.RateLimit,
		hashKey:             cfg.Key,
		client:              resty.New(),
		log:                 log,
		sendQueue:           make(chan reportTask, cfg.RateLimit),
	}, nil
}

// Run запускает независимые горутины:
// сбор runtime-метрик, сбор системных метрик и отправку метрик на Сервер.
func (a *Agent) Run(ctx context.Context) {
	go a.runRuntimePollLoop(ctx)
	go a.runSystemPollLoop(ctx)

	for i := 0; i < a.rateLimit; i++ {
		go a.runSendWorker(ctx)
	}

	a.runReportLoop(ctx)
}

func (a *Agent) runRuntimePollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollRuntime()
		}
	}
}
func (a *Agent) runSystemPollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	defer ticker.Stop()

	// NOTE: Первый вызов нужен, чтобы инициализировать базу для cpu.Percent(0, true).
	// https://pkg.go.dev/github.com/shirou/gopsutil/v4/cpu
	if _, err := cpu.Percent(0, true); err != nil {
		a.logError(fmt.Errorf("init cpu percent: %w", err))
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollSystem()
		}
	}
}

func (a *Agent) runReportLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.reportIntervalInSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.report()
		}
	}
}

func (a *Agent) runSendWorker(ctx context.Context) {
	sendTasks := (<-chan reportTask)(a.sendQueue)

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-sendTasks:
			if !ok {
				return
			}
			if err := a.sendMetrics(ctx, task.metrics); err != nil {
				// Пока задача отправлялась, runtime-сборщик уже мог накопить
				// новые poll'ы, поэтому возвращаем старое значение через Add.
				a.pollSinceReport.Add(task.pollCount)
				a.logError(err)
			}
		}
	}
}

func (a *Agent) logError(err error) {
	if err == nil {
		return
	}
	a.log.Error("agent error", zap.Error(err))
}
