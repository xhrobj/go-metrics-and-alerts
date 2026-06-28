package agent

import (
	"context"

	"github.com/xhrobj/go-metrics-and-alerts/internal/agent/service"
)

type reportingServiceStub struct {
	initSystemPollFunc        func()
	pollRuntimeFunc           func()
	pollSystemFunc            func()
	preparePeriodicReportFunc func(context.Context) (service.Report, bool, error)
	sendFunc                  func(context.Context, service.Report) error
	restoreFunc               func(service.Report)
	flushFunc                 func(context.Context) error
}

func (s *reportingServiceStub) InitSystemPoll() {
	if s.initSystemPollFunc != nil {
		s.initSystemPollFunc()
	}
}

func (s *reportingServiceStub) PollRuntime() {
	if s.pollRuntimeFunc != nil {
		s.pollRuntimeFunc()
	}
}

func (s *reportingServiceStub) PollSystem() {
	if s.pollSystemFunc != nil {
		s.pollSystemFunc()
	}
}

func (s *reportingServiceStub) PreparePeriodicReport(
	ctx context.Context,
) (service.Report, bool, error) {
	if s.preparePeriodicReportFunc != nil {
		return s.preparePeriodicReportFunc(ctx)
	}

	return service.Report{}, false, nil
}

func (s *reportingServiceStub) Send(
	ctx context.Context,
	report service.Report,
) error {
	if s.sendFunc != nil {
		return s.sendFunc(ctx, report)
	}

	return nil
}

func (s *reportingServiceStub) Restore(report service.Report) {
	if s.restoreFunc != nil {
		s.restoreFunc(report)
	}
}

func (s *reportingServiceStub) Flush(ctx context.Context) error {
	if s.flushFunc != nil {
		return s.flushFunc(ctx)
	}

	return nil
}
