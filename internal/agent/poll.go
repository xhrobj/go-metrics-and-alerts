package agent

import (
	"fmt"
	"math/rand"
	"runtime"
)

func (a *Agent) poll() {
	fmt.Printf("%d poll\n", a.uptime)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	a.repo.UpdateGauge("Alloc", float64(ms.Alloc))
	a.repo.UpdateGauge("BuckHashSys", float64(ms.BuckHashSys))
	a.repo.UpdateGauge("Frees", float64(ms.Frees))
	a.repo.UpdateGauge("GCCPUFraction", float64(ms.GCCPUFraction))
	a.repo.UpdateGauge("GCSys", float64(ms.GCSys))
	a.repo.UpdateGauge("HeapAlloc", float64(ms.HeapAlloc))
	a.repo.UpdateGauge("HeapIdle", float64(ms.HeapIdle))
	a.repo.UpdateGauge("HeapInuse", float64(ms.HeapInuse))
	a.repo.UpdateGauge("HeapObjects", float64(ms.HeapObjects))
	a.repo.UpdateGauge("HeapReleased", float64(ms.HeapReleased))
	a.repo.UpdateGauge("HeapSys", float64(ms.HeapSys))
	a.repo.UpdateGauge("LastGC", float64(ms.LastGC))
	a.repo.UpdateGauge("Lookups", float64(ms.Lookups))
	a.repo.UpdateGauge("MCacheInuse", float64(ms.MCacheInuse))
	a.repo.UpdateGauge("MCacheSys", float64(ms.MCacheSys))
	a.repo.UpdateGauge("MSpanInuse", float64(ms.MSpanInuse))
	a.repo.UpdateGauge("MSpanSys", float64(ms.MSpanSys))
	a.repo.UpdateGauge("Mallocs", float64(ms.Mallocs))
	a.repo.UpdateGauge("NextGC", float64(ms.NextGC))
	a.repo.UpdateGauge("NumForcedGC", float64(ms.NumForcedGC))
	a.repo.UpdateGauge("NumGC", float64(ms.NumGC))
	a.repo.UpdateGauge("OtherSys", float64(ms.OtherSys))
	a.repo.UpdateGauge("PauseTotalNs", float64(ms.PauseTotalNs))
	a.repo.UpdateGauge("StackInuse", float64(ms.StackInuse))
	a.repo.UpdateGauge("StackSys", float64(ms.StackSys))
	a.repo.UpdateGauge("Sys", float64(ms.Sys))
	a.repo.UpdateGauge("TotalAlloc", float64(ms.TotalAlloc))

	a.repo.UpdateGauge("RandomValue", rand.Float64())
	a.repo.UpdateCounter("PollCount", 1)
}
