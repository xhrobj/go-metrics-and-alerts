package agent

import (
	"fmt"
	"time"
)

var uptime uint = 0

func Run() {
	for {
		if uptime%2 == 0 {
			poll()
		}
		if uptime%10 == 0 {
			report()
		}
		time.Sleep(1 * time.Second)
		uptime++
	}
}

func report() {
	fmt.Printf("%d report\n", uptime)
}
