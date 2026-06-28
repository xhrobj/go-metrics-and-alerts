package agent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/service"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"go.uber.org/zap"
)

func TestNewReturnsErrorForNilReportingService(t *testing.T) {
	cfg := config.AgentConfig{
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}

	_, err := New(nil, cfg, zap.NewNop())
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}

	if got, want := err.Error(), "reporting service is nil"; got != want {
		t.Fatalf("New() error = %q, want %q", got, want)
	}
}

func TestNewReturnsErrorForTypedNilReportingService(t *testing.T) {
	cfg := config.AgentConfig{
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           5,
	}

	var reportingService *reportingServiceStub

	_, err := New(reportingService, cfg, zap.NewNop())
	if err == nil {
		t.Fatal("New() error = nil, want error")
	}

	if got, want := err.Error(), "reporting service is nil"; got != want {
		t.Fatalf("New() error = %q, want %q", got, want)
	}
}

// TestRunDrainsQueueBeforeFlush проверяет, что при shutdown Агент
// дожидается активной отправки, опустошает очередь и только затем вызывает Flush.
func TestRunDrainsQueueBeforeFlush(t *testing.T) {
	firstSendStarted := make(chan struct{})
	allowFirstSendToFinish := make(chan struct{})
	flushCalled := make(chan struct{})

	var sendCount atomic.Int32
	report := service.Report{Metrics: []model.Metrics{{ID: "Alloc"}}, PollCount: 42}

	service := &reportingServiceStub{
		preparePeriodicReportFunc: func(context.Context) (service.Report, bool, error) {
			return report, true, nil
		},
		sendFunc: func(_ context.Context, got service.Report) error {
			if got.PollCount != report.PollCount {
				t.Errorf("Send() pollCount = %d, want %d", got.PollCount, report.PollCount)
			}
			if sendCount.Add(1) == 1 {
				close(firstSendStarted)
				<-allowFirstSendToFinish
			}
			return nil
		},
		flushFunc: func(context.Context) error {
			close(flushCalled)
			return nil
		},
	}

	t.Cleanup(func() {
		select {
		case <-allowFirstSendToFinish:
		default:
			close(allowFirstSendToFinish)
		}
	})

	a, err := New(service, config.AgentConfig{
		PollIntervalInSec:   3600,
		ReportIntervalInSec: 3600,
		RateLimit:           1,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	a.report()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- a.Run(ctx)
	}()

	select {
	case <-firstSendStarted:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for active send")
	}

	cancel()

	select {
	case err := <-runErrCh:
		t.Fatalf("Run() returned before active send completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-flushCalled:
		t.Fatal("Flush() called before active send completed")
	default:
	}

	close(allowFirstSendToFinish)

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for Run() to stop")
	}

	select {
	case <-flushCalled:
	default:
		t.Fatal("Flush() was not called")
	}

	if got, want := sendCount.Load(), int32(1); got != want {
		t.Fatalf("send count = %d, want %d", got, want)
	}
}
