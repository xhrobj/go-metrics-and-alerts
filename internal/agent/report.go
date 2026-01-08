package agent

import "fmt"

func (a *Agent) report() {
	gauges, counters := a.repo.Snapshot()

	fmt.Printf("%d >>> report\n\n", a.uptime)

	fmt.Printf("%v\n\n", gauges)
	fmt.Printf("%v\n\n", counters)
}
