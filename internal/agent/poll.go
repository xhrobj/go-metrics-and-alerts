package agent

import (
	"log"
	"math/rand"
	"runtime"
)

func (a *Agent) poll() {
	log.Printf("poll (%d)", a.pollSinceReport)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	_ = a.repo.UpdateGauge("Alloc", float64(ms.Alloc))
	_ = a.repo.UpdateGauge("BuckHashSys", float64(ms.BuckHashSys))
	_ = a.repo.UpdateGauge("Frees", float64(ms.Frees))
	_ = a.repo.UpdateGauge("GCCPUFraction", float64(ms.GCCPUFraction))
	_ = a.repo.UpdateGauge("GCSys", float64(ms.GCSys))
	_ = a.repo.UpdateGauge("HeapAlloc", float64(ms.HeapAlloc))
	_ = a.repo.UpdateGauge("HeapIdle", float64(ms.HeapIdle))
	_ = a.repo.UpdateGauge("HeapInuse", float64(ms.HeapInuse))
	_ = a.repo.UpdateGauge("HeapObjects", float64(ms.HeapObjects))
	_ = a.repo.UpdateGauge("HeapReleased", float64(ms.HeapReleased))
	_ = a.repo.UpdateGauge("HeapSys", float64(ms.HeapSys))
	_ = a.repo.UpdateGauge("LastGC", float64(ms.LastGC))
	_ = a.repo.UpdateGauge("Lookups", float64(ms.Lookups))
	_ = a.repo.UpdateGauge("MCacheInuse", float64(ms.MCacheInuse))
	_ = a.repo.UpdateGauge("MCacheSys", float64(ms.MCacheSys))
	_ = a.repo.UpdateGauge("MSpanInuse", float64(ms.MSpanInuse))
	_ = a.repo.UpdateGauge("MSpanSys", float64(ms.MSpanSys))
	_ = a.repo.UpdateGauge("Mallocs", float64(ms.Mallocs))
	_ = a.repo.UpdateGauge("NextGC", float64(ms.NextGC))
	_ = a.repo.UpdateGauge("NumForcedGC", float64(ms.NumForcedGC))
	_ = a.repo.UpdateGauge("NumGC", float64(ms.NumGC))
	_ = a.repo.UpdateGauge("OtherSys", float64(ms.OtherSys))
	_ = a.repo.UpdateGauge("PauseTotalNs", float64(ms.PauseTotalNs))
	_ = a.repo.UpdateGauge("StackInuse", float64(ms.StackInuse))
	_ = a.repo.UpdateGauge("StackSys", float64(ms.StackSys))
	_ = a.repo.UpdateGauge("Sys", float64(ms.Sys))
	_ = a.repo.UpdateGauge("TotalAlloc", float64(ms.TotalAlloc))

	_ = a.repo.UpdateGauge("RandomValue", rand.Float64())

	a.pollSinceReport++
}
