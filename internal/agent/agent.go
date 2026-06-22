package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/config"
	"go.uber.org/zap"
)

const shutdownTimeout = time.Second * 30

// Agent управляет runtime Агента: ticker'ами, очередью отправки,
// worker'ами и graceful shutdown.
type Agent struct {
	service             *ReportingService
	pollIntervalInSec   int
	reportIntervalInSec int
	rateLimit           int
	log                 *zap.Logger

	// sendQueue используется report() для постановки задач на отправку.
	sendQueue chan<- reportTask
	// recvQueue используется send worker'ами для чтения задач из той же очереди.
	recvQueue <-chan reportTask
}

// New создаёт нового Агента с указанными параметрами конфигурации.
func New(service *ReportingService, cfg config.AgentConfig, log *zap.Logger) (*Agent, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if service == nil {
		return nil, fmt.Errorf("reporting service is nil")
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

	queue := make(chan reportTask, cfg.RateLimit)

	return &Agent{
		service:             service,
		pollIntervalInSec:   cfg.PollIntervalInSec,
		reportIntervalInSec: cfg.ReportIntervalInSec,
		rateLimit:           cfg.RateLimit,
		log:                 log,
		sendQueue:           queue,
		recvQueue:           queue,
	}, nil
}

// Run запускает независимые горутины сбора и отправки метрик.
//
// При отмене контекста ждёт завершения активных операций и
// отправляет финальный снимок.
func (a *Agent) Run(ctx context.Context) error {
	sendCtx, cancelSend := context.WithCancel(context.WithoutCancel(ctx))
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

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.WithoutCancel(ctx),
		shutdownTimeout,
	)
	defer cancelShutdown()

	if err := waitForGroup(shutdownCtx, &pollAndReportWG); err != nil {
		cancelSend()
		return fmt.Errorf("wait poll and report loops: %w", err)
	}

	close(a.sendQueue)

	if err := waitForGroup(shutdownCtx, &sendWG); err != nil {
		cancelSend()
		return fmt.Errorf("wait send workers: %w", err)
	}

	if err := a.service.flush(shutdownCtx); err != nil {
		return err
	}

	a.log.Info("agent stopped")

	return nil
}

func waitForGroup(ctx context.Context, wg *sync.WaitGroup) error {
	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *Agent) runRuntimePollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.service.pollRuntime()
		}
	}
}

func (a *Agent) runSystemPollLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.pollIntervalInSec) * time.Second)
	defer ticker.Stop()

	a.service.initSystemPoll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.service.pollSystem()
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
			if err := a.service.send(ctx, task); err != nil {
				a.log.Error("send metrics failed",
					zap.Int64("pollCount", task.pollCount),
					zap.Error(err),
				)
			}
		}
	}
}
