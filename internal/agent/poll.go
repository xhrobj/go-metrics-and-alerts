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

func (a *Agent) pollRuntime() {
	ctx := context.Background()

	a.log.Info("poll runtime",
		zap.Int64("pollSinceReport", a.pollSinceReport.Load()),
	)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	a.updateGauge(ctx, "Alloc", float64(ms.Alloc))
	a.updateGauge(ctx, "BuckHashSys", float64(ms.BuckHashSys))
	a.updateGauge(ctx, "Frees", float64(ms.Frees))
	a.updateGauge(ctx, "GCCPUFraction", float64(ms.GCCPUFraction))
	a.updateGauge(ctx, "GCSys", float64(ms.GCSys))
	a.updateGauge(ctx, "HeapAlloc", float64(ms.HeapAlloc))
	a.updateGauge(ctx, "HeapIdle", float64(ms.HeapIdle))
	a.updateGauge(ctx, "HeapInuse", float64(ms.HeapInuse))
	a.updateGauge(ctx, "HeapObjects", float64(ms.HeapObjects))
	a.updateGauge(ctx, "HeapReleased", float64(ms.HeapReleased))
	a.updateGauge(ctx, "HeapSys", float64(ms.HeapSys))
	a.updateGauge(ctx, "LastGC", float64(ms.LastGC))
	a.updateGauge(ctx, "Lookups", float64(ms.Lookups))
	a.updateGauge(ctx, "MCacheInuse", float64(ms.MCacheInuse))
	a.updateGauge(ctx, "MCacheSys", float64(ms.MCacheSys))
	a.updateGauge(ctx, "MSpanInuse", float64(ms.MSpanInuse))
	a.updateGauge(ctx, "MSpanSys", float64(ms.MSpanSys))
	a.updateGauge(ctx, "Mallocs", float64(ms.Mallocs))
	a.updateGauge(ctx, "NextGC", float64(ms.NextGC))
	a.updateGauge(ctx, "NumForcedGC", float64(ms.NumForcedGC))
	a.updateGauge(ctx, "NumGC", float64(ms.NumGC))
	a.updateGauge(ctx, "OtherSys", float64(ms.OtherSys))
	a.updateGauge(ctx, "PauseTotalNs", float64(ms.PauseTotalNs))
	a.updateGauge(ctx, "StackInuse", float64(ms.StackInuse))
	a.updateGauge(ctx, "StackSys", float64(ms.StackSys))
	a.updateGauge(ctx, "Sys", float64(ms.Sys))
	a.updateGauge(ctx, "TotalAlloc", float64(ms.TotalAlloc))

	a.updateGauge(ctx, "RandomValue", rand.Float64())

	a.pollSinceReport.Add(1)
}

func (a *Agent) pollSystem() {
	ctx := context.Background()

	a.log.Info("poll system")

	vm, err := mem.VirtualMemory()
	if err != nil {
		a.logError(fmt.Errorf("read virtual memory: %w", err))
		return
	}

	a.updateGauge(ctx, "TotalMemory", float64(vm.Total))
	a.updateGauge(ctx, "FreeMemory", float64(vm.Free))

	cpuPercents, err := cpu.Percent(0, true)
	if err != nil {
		a.logError(fmt.Errorf("read cpu percent: %w", err))
		return
	}

	for i, percent := range cpuPercents {
		name := fmt.Sprintf("CPUutilization%d", i+1)
		a.updateGauge(ctx, name, percent)
	}
}

func (a *Agent) updateGauge(ctx context.Context, name string, value float64) {
	if err := a.repo.UpdateGauge(ctx, name, value); err != nil {
		a.logError(err)
	}
}
