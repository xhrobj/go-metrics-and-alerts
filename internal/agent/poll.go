package agent

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
)

func (s *ReportingService) initSystemPoll() {
	// NOTE: Первый вызов нужен, чтобы инициализировать базу для cpu.Percent(0, true).
	// https://pkg.go.dev/github.com/shirou/gopsutil/v4/cpu
	if _, err := cpu.Percent(0, true); err != nil {
		s.log.Error("init cpu percent", zap.Error(err))
	}
}

func (s *ReportingService) pollRuntime() {
	ctx := context.Background()

	s.log.Info("poll runtime",
		zap.Int64("pollSinceReport", s.pollSinceReport.Load()),
	)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	s.updateGauge(ctx, "Alloc", float64(ms.Alloc))
	s.updateGauge(ctx, "BuckHashSys", float64(ms.BuckHashSys))
	s.updateGauge(ctx, "Frees", float64(ms.Frees))
	s.updateGauge(ctx, "GCCPUFraction", float64(ms.GCCPUFraction))
	s.updateGauge(ctx, "GCSys", float64(ms.GCSys))
	s.updateGauge(ctx, "HeapAlloc", float64(ms.HeapAlloc))
	s.updateGauge(ctx, "HeapIdle", float64(ms.HeapIdle))
	s.updateGauge(ctx, "HeapInuse", float64(ms.HeapInuse))
	s.updateGauge(ctx, "HeapObjects", float64(ms.HeapObjects))
	s.updateGauge(ctx, "HeapReleased", float64(ms.HeapReleased))
	s.updateGauge(ctx, "HeapSys", float64(ms.HeapSys))
	s.updateGauge(ctx, "LastGC", float64(ms.LastGC))
	s.updateGauge(ctx, "Lookups", float64(ms.Lookups))
	s.updateGauge(ctx, "MCacheInuse", float64(ms.MCacheInuse))
	s.updateGauge(ctx, "MCacheSys", float64(ms.MCacheSys))
	s.updateGauge(ctx, "MSpanInuse", float64(ms.MSpanInuse))
	s.updateGauge(ctx, "MSpanSys", float64(ms.MSpanSys))
	s.updateGauge(ctx, "Mallocs", float64(ms.Mallocs))
	s.updateGauge(ctx, "NextGC", float64(ms.NextGC))
	s.updateGauge(ctx, "NumForcedGC", float64(ms.NumForcedGC))
	s.updateGauge(ctx, "NumGC", float64(ms.NumGC))
	s.updateGauge(ctx, "OtherSys", float64(ms.OtherSys))
	s.updateGauge(ctx, "PauseTotalNs", float64(ms.PauseTotalNs))
	s.updateGauge(ctx, "StackInuse", float64(ms.StackInuse))
	s.updateGauge(ctx, "StackSys", float64(ms.StackSys))
	s.updateGauge(ctx, "Sys", float64(ms.Sys))
	s.updateGauge(ctx, "TotalAlloc", float64(ms.TotalAlloc))

	s.updateGauge(ctx, "RandomValue", rand.Float64())

	s.pollSinceReport.Add(1)
}

func (s *ReportingService) pollSystem() {
	ctx := context.Background()

	s.log.Info("poll system")

	vm, err := mem.VirtualMemory()
	if err != nil {
		s.log.Error("read virtual memory", zap.Error(err))
		return
	}

	s.updateGauge(ctx, "TotalMemory", float64(vm.Total))
	s.updateGauge(ctx, "FreeMemory", float64(vm.Free))

	cpuPercents, err := cpu.Percent(0, true)
	if err != nil {
		s.log.Error("read cpu percent", zap.Error(err))
		return
	}

	for i, percent := range cpuPercents {
		name := fmt.Sprintf("CPUutilization%d", i+1)
		s.updateGauge(ctx, name, percent)
	}
}

func (s *ReportingService) updateGauge(ctx context.Context, name string, value float64) {
	if err := s.repo.UpdateGauge(ctx, name, value); err != nil {
		s.log.Error("update gauge failed",
			zap.String("metric", name),
			zap.Error(err),
		)
	}
}
