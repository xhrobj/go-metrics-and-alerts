package agent

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"go.uber.org/zap"
)

const shutdownTimeout = time.Second * 30 // мм но по факту ожидается после ожидания сборщиков ...

// AgentStorage описывает хранилище метрик, используемое Агентом.
type AgentStorage interface {
	UpdateGauge(context.Context, string, float64) error
	Snapshot(context.Context) (map[string]float64, map[string]int64, error)
}

type reportTask struct {
	metrics   []model.Metrics
	pollCount int64
}

// Agent собирает системные и runtime-метрики и отправляет их на Сервер по HTTP.
type Agent struct {
	repo                AgentStorage
	baseURL             string
	pollIntervalInSec   int
	reportIntervalInSec int
	rateLimit           int
	hashKey             string
	publicKey           *rsa.PublicKey
	client              *resty.Client
	log                 *zap.Logger

	// sendQueue используется report() для постановки задач на отправку.
	sendQueue chan<- reportTask
	// recvQueue используется send worker'ами для чтения задач из той же очереди.
	recvQueue <-chan reportTask

	// pollSinceReport - количество вызовов pollRuntime() с момента последней
	// успешной отправки отчёта. Используется для формирования метрики PollCount.
	pollSinceReport atomic.Int64
}

// New создаёт нового Агента с указанными параметрами конфигурации.
func New(repo AgentStorage, cfg config.AgentConfig, log *zap.Logger) (*Agent, error) {
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

	var publicKey *rsa.PublicKey
	if cfg.CryptoKey != "" {
		loadedPublicKey, err := encryption.LoadPublicKey(cfg.CryptoKey)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}

		publicKey = loadedPublicKey
	}

	baseURL := cfg.ServerAddr
	if !strings.Contains(baseURL, "://") {
		baseURL = "http://" + baseURL
	}

	queue := make(chan reportTask, cfg.RateLimit)

	return &Agent{
		repo:                repo,
		baseURL:             baseURL,
		pollIntervalInSec:   cfg.PollIntervalInSec,
		reportIntervalInSec: cfg.ReportIntervalInSec,
		rateLimit:           cfg.RateLimit,
		hashKey:             cfg.Key,
		publicKey:           publicKey,
		client:              resty.New(),
		log:                 log,
		sendQueue:           queue,
		recvQueue:           queue,
	}, nil
}

// Run запускает независимые горутины:
// сбор runtime-метрик, сбор системных метрик и отправку метрик на Сервер.
//
// При отмене контекста ждёт завершения активных операций и
//
//	отправляет финальный снимок.
func (a *Agent) Run(ctx context.Context) error {
	// контект для отправки для воркеров; сборщики будут работать с внешим ctx
	sendCtx, cancelSend := context.WithCancel(context.Background())
	defer cancelSend()

	var pollAndReportWG sync.WaitGroup
	pollAndReportWG.Add(3)

	go func() {
		defer pollAndReportWG.Done()
		a.runRuntimePollLoop(ctx)
	}()

	go func() {
		defer pollAndReportWG.Done()
		a.runSystemPollLoop(ctx)
	}()

	go func() {
		defer pollAndReportWG.Done()
		a.runReportLoop(ctx)
	}()

	var sendWG sync.WaitGroup
	for i := 0; i < a.rateLimit; i++ {
		sendWG.Add(1)

		go func() {
			defer sendWG.Done()
			a.runSendWorker(sendCtx)
		}()
	}

	<-ctx.Done()
	a.log.Info("shutdown signal received")

	// притормозим shutdown и сначала дожидаемся остановики сборщиков и report loop,
	// чтобы никто больше не записывал метрики и не отправлял задачи в очередь
	pollAndReportWG.Wait()
	close(a.sendQueue)

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancelShutdown()

	sendDone := make(chan struct{})
	go func() {
		sendWG.Wait()
		close(sendDone)
	}()

	select {
	case <-sendDone:
	case <-shutdownCtx.Done():
		cancelSend()
		<-sendDone

		return fmt.Errorf("wait send workers: %w", shutdownCtx.Err())
	}

	if err := a.flush(shutdownCtx); err != nil {
		return err
	}

	a.log.Info("agent stopped")

	return nil
}

func (a *Agent) flush(ctx context.Context) error {
	pollCount := a.pollSinceReport.Swap(0)

	metrics, err := a.buildMetricsBatch(pollCount)
	if err != nil {
		a.pollSinceReport.Add(pollCount)
		return fmt.Errorf("build final metrics batch: %w", err)
	}

	if err := a.sendMetrics(ctx, metrics); err != nil {
		a.pollSinceReport.Add(pollCount)
		return fmt.Errorf("send final metrics batch: %w", err)
	}

	return nil
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
		a.log.Error("init cpu percent", zap.Error(err))
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
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-a.recvQueue:
			if !ok {
				return
			}
			if err := a.sendMetrics(ctx, task.metrics); err != nil {
				// Пока задача отправлялась, runtime-сборщик уже мог накопить
				// новые poll'ы, поэтому возвращаем старое значение через Add.
				a.pollSinceReport.Add(task.pollCount)
				a.log.Error("send metrics failed",
					zap.Int64("pollCount", task.pollCount),
					zap.Error(err),
				)
			}
		}
	}
}
