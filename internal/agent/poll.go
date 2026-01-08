package agent

import (
	"fmt"
	"runtime"
)

func (a *Agent) poll() {
	fmt.Printf("%d poll\n", a.uptime)

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	fmt.Printf(
		"\nAlloc=%d HeapAlloc=%d HeapObjects=%d NumGC=%d GCCPUFraction=%.4f PauseTotalNs=%d\n\n",
		ms.Alloc,
		ms.HeapAlloc,
		ms.HeapObjects,
		ms.NumGC,
		ms.GCCPUFraction,
		ms.PauseTotalNs,
	)
}
