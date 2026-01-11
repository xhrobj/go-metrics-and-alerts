package main

import (
	"github.com/xhrobj/go-metrics-and-alerts/internal/agent"
	"github.com/xhrobj/go-metrics-and-alerts/internal/repository"
)

func main() {
	parseFlags()

	repo := repository.NewMemStorage()
	a := agent.New(
		repo,
		"http://"+flagServerAddr,
		flagPollIntervalInSec,
		flagReportIntervalInSec)

	a.Run()
}
