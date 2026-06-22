package agent

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/config"
	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/service"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
	"go.uber.org/zap"
)

// TestAgentReportQueuesPreparedReport проверяет границу Agent -> ReportingService:
// runtime ставит подготовленный сервисом отчёт в очередь отправки.
func TestAgentReportQueuesPreparedReport(t *testing.T) {
	wantReport := service.Report{Metrics: []model.Metrics{{ID: "Alloc"}}, PollCount: 3}
	service := &reportingServiceStub{
		preparePeriodicReportFunc: func(context.Context) (service.Report, bool, error) {
			return wantReport, true, nil
		},
	}

	a, err := New(service, config.AgentConfig{
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           1,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	a.report()

	select {
	case gotReport := <-a.recvQueue:
		if got, want := gotReport.PollCount, wantReport.PollCount; got != want {
			t.Fatalf("PollCount = %d, want %d", got, want)
		}
		if got, want := gotReport.Metrics[0].ID, "Alloc"; got != want {
			t.Fatalf("metric ID = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for report")
	}
}

// TestAgentReportRestoresReportWhenQueueIsFull проверяет,
// что runtime возвращает сервису отчёт, не принятый заполненной очередью.
func TestAgentReportRestoresReportWhenQueueIsFull(t *testing.T) {
	reports := []service.Report{
		{PollCount: 1},
		{PollCount: 2},
	}
	prepareCall := 0
	restored := make(chan service.Report, 1)

	service := &reportingServiceStub{
		preparePeriodicReportFunc: func(context.Context) (service.Report, bool, error) {
			report := reports[prepareCall]
			prepareCall++
			return report, true, nil
		},
		restoreFunc: func(report service.Report) {
			restored <- report
		},
	}

	a, err := New(service, config.AgentConfig{
		PollIntervalInSec:   2,
		ReportIntervalInSec: 10,
		RateLimit:           1,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	a.report()
	a.report()

	select {
	case gotReport := <-restored:
		if got, want := gotReport.PollCount, int64(2); got != want {
			t.Fatalf("restored PollCount = %d, want %d", got, want)
		}
	default:
		t.Fatal("Restore() was not called")
	}
}
