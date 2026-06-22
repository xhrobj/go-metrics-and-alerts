package agent

import (
	"context"
	"fmt"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

func (a *Agent) report() {
	task, ok, err := a.service.preparePeriodicReport(context.Background())
	if err != nil || !ok {
		return
	}

	select {
	case a.sendQueue <- task:
	default:
		a.service.restore(task)
		a.log.Warn("sendQueue is full")
	}
}

func (s *ReportingService) preparePeriodicReport(ctx context.Context) (reportTask, bool, error) {
	return s.prepareReport(ctx, true)
}

func (s *ReportingService) prepareFinalReport(ctx context.Context) (reportTask, error) {
	task, _, err := s.prepareReport(ctx, false)
	return task, err
}

func (s *ReportingService) prepareReport(
	ctx context.Context,
	skipWithoutPolls bool,
) (reportTask, bool, error) {
	// Запоминаем значение и обнуляем накопитель.
	pollCount := s.pollSinceReport.Swap(0)
	if skipWithoutPolls && pollCount == 0 {
		return reportTask{}, false, nil
	}

	metrics, err := s.buildMetricsBatch(ctx, pollCount)
	if err != nil {
		// Runtime-сборщик мог уже увеличить счётчик, поэтому возвращаем
		// старое значение через Add, а не через Store.
		s.pollSinceReport.Add(pollCount)
		return reportTask{}, false, err
	}

	if len(metrics) == 0 {
		s.pollSinceReport.Add(pollCount)
		return reportTask{}, false, nil
	}

	return reportTask{
		metrics:   metrics,
		pollCount: pollCount,
	}, true, nil
}

func (s *ReportingService) buildMetricsBatch(
	ctx context.Context,
	pollCount int64,
) ([]model.Metrics, error) {
	gauges, _, err := s.repo.Snapshot(ctx)
	if err != nil {
		return nil, fmt.Errorf("snapshot metrics: %w", err)
	}

	// Snapshot сейчас не гарантирует одномоментную согласованность всех метрик:
	// часть gauge-метрик может быть уже обновлена другой горутиной в момент
	// формирования batch.
	metrics := make([]model.Metrics, 0, len(gauges)+1)

	for name, value := range gauges {
		v := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}

	metrics = append(metrics, model.Metrics{
		ID:    "PollCount",
		MType: model.Counter,
		Delta: &pollCount,
	})

	return metrics, nil
}

func (s *ReportingService) send(ctx context.Context, task reportTask) error {
	if err := s.sender.Send(ctx, task.metrics); err != nil {
		s.restore(task)
		return err
	}

	return nil
}

func (s *ReportingService) restore(task reportTask) {
	// Пока задача находилась в очереди или отправлялась, runtime-сборщик мог
	// накопить новые poll'ы, поэтому возвращаем старое значение через Add.
	s.pollSinceReport.Add(task.pollCount)
}

func (s *ReportingService) flush(ctx context.Context) error {
	task, err := s.prepareFinalReport(ctx)
	if err != nil {
		return fmt.Errorf("build final metrics batch: %w", err)
	}

	if err := s.send(ctx, task); err != nil {
		return fmt.Errorf("send final metrics batch: %w", err)
	}

	return nil
}
