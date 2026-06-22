package service

import (
	"context"
	"fmt"

	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

// PreparePeriodicReport формирует очередной отчёт.
// Если после предыдущего отчёта не было runtime-опросов, ok будет false.
func (s *ReportingService) PreparePeriodicReport(ctx context.Context) (Report, bool, error) {
	return s.prepareReport(ctx, true)
}

func (s *ReportingService) prepareFinalReport(ctx context.Context) (Report, error) {
	report, _, err := s.prepareReport(ctx, false)
	return report, err
}

func (s *ReportingService) prepareReport(
	ctx context.Context,
	skipWithoutPolls bool,
) (Report, bool, error) {
	// Запоминаем значение и обнуляем накопитель.
	pollCount := s.pollSinceReport.Swap(0)
	if skipWithoutPolls && pollCount == 0 {
		return Report{}, false, nil
	}

	metrics, err := s.buildMetricsBatch(ctx, pollCount)
	if err != nil {
		// Runtime-сборщик мог уже увеличить счётчик, поэтому возвращаем
		// старое значение через Add, а не через Store.
		s.pollSinceReport.Add(pollCount)
		return Report{}, false, err
	}

	if len(metrics) == 0 {
		s.pollSinceReport.Add(pollCount)
		return Report{}, false, nil
	}

	return Report{Metrics: metrics, PollCount: pollCount}, true, nil
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

// Send отправляет подготовленный отчёт и восстанавливает его PollCount
// при ошибке транспорта.
func (s *ReportingService) Send(ctx context.Context, report Report) error {
	if err := s.sender.Send(ctx, report.Metrics); err != nil {
		s.Restore(report)
		return err
	}

	return nil
}

// Restore возвращает PollCount неотправленного отчёта в накопитель.
func (s *ReportingService) Restore(report Report) {
	// Пока задача находилась в очереди или отправлялась, runtime-сборщик мог
	// накопить новые poll'ы, поэтому возвращаем старое значение через Add.
	s.pollSinceReport.Add(report.PollCount)
}

// Flush формирует и синхронно отправляет финальный отчёт.
func (s *ReportingService) Flush(ctx context.Context) error {
	report, err := s.prepareFinalReport(ctx)
	if err != nil {
		return fmt.Errorf("build final metrics batch: %w", err)
	}

	if err := s.Send(ctx, report); err != nil {
		return fmt.Errorf("send final metrics batch: %w", err)
	}

	return nil
}
