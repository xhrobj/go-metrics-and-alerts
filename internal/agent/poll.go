package agent

import (
	"context"
	"log"
	"math/rand"
	"runtime"
)

func (a *Agent) poll() {
	log.Printf("poll (%d)", a.pollSinceReport)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	ctx := context.Background()

	_ = a.repo.UpdateGauge(ctx, "Alloc", float64(ms.Alloc))
	_ = a.repo.UpdateGauge(ctx, "BuckHashSys", float64(ms.BuckHashSys))
	_ = a.repo.UpdateGauge(ctx, "Frees", float64(ms.Frees))
	_ = a.repo.UpdateGauge(ctx, "GCCPUFraction", float64(ms.GCCPUFraction))
	_ = a.repo.UpdateGauge(ctx, "GCSys", float64(ms.GCSys))
	_ = a.repo.UpdateGauge(ctx, "HeapAlloc", float64(ms.HeapAlloc))
	_ = a.repo.UpdateGauge(ctx, "HeapIdle", float64(ms.HeapIdle))
	_ = a.repo.UpdateGauge(ctx, "HeapInuse", float64(ms.HeapInuse))
	_ = a.repo.UpdateGauge(ctx, "HeapObjects", float64(ms.HeapObjects))
	_ = a.repo.UpdateGauge(ctx, "HeapReleased", float64(ms.HeapReleased))
	_ = a.repo.UpdateGauge(ctx, "HeapSys", float64(ms.HeapSys))
	_ = a.repo.UpdateGauge(ctx, "LastGC", float64(ms.LastGC))
	_ = a.repo.UpdateGauge(ctx, "Lookups", float64(ms.Lookups))
	_ = a.repo.UpdateGauge(ctx, "MCacheInuse", float64(ms.MCacheInuse))
	_ = a.repo.UpdateGauge(ctx, "MCacheSys", float64(ms.MCacheSys))
	_ = a.repo.UpdateGauge(ctx, "MSpanInuse", float64(ms.MSpanInuse))
	_ = a.repo.UpdateGauge(ctx, "MSpanSys", float64(ms.MSpanSys))
	_ = a.repo.UpdateGauge(ctx, "Mallocs", float64(ms.Mallocs))
	_ = a.repo.UpdateGauge(ctx, "NextGC", float64(ms.NextGC))
	_ = a.repo.UpdateGauge(ctx, "NumForcedGC", float64(ms.NumForcedGC))
	_ = a.repo.UpdateGauge(ctx, "NumGC", float64(ms.NumGC))
	_ = a.repo.UpdateGauge(ctx, "OtherSys", float64(ms.OtherSys))
	_ = a.repo.UpdateGauge(ctx, "PauseTotalNs", float64(ms.PauseTotalNs))
	_ = a.repo.UpdateGauge(ctx, "StackInuse", float64(ms.StackInuse))
	_ = a.repo.UpdateGauge(ctx, "StackSys", float64(ms.StackSys))
	_ = a.repo.UpdateGauge(ctx, "Sys", float64(ms.Sys))
	_ = a.repo.UpdateGauge(ctx, "TotalAlloc", float64(ms.TotalAlloc))

	_ = a.repo.UpdateGauge(ctx, "RandomValue", rand.Float64())

	// !!!: теперь тут атомарный инкремент
	a.pollSinceReport.Add(1)
}
