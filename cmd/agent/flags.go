package main

import (
	"flag"
)

const (
	serverAddrDefault          = "localhost:8080"
	pollIntervalInSecDefault   = 2
	reportIntervalInSecDefault = 10
)

var (
	// адрес HTTP-сервера для приёма метрик
	flagServerAddr string
	// интервал опроса метрик
	flagPollIntervalInSec int
	// частота отправки метрик на сервер
	flagReportIntervalInSec int
)

// обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagServerAddr, "a", serverAddrDefault, "address of the HTTP server (host:port)")
	flag.IntVar(&flagPollIntervalInSec, "p", pollIntervalInSecDefault, "runtime metrics polling interval in seconds")
	flag.IntVar(&flagReportIntervalInSec, "r", reportIntervalInSecDefault, "metrics reporting interval in seconds")

	flag.Parse()
	validateFlags()
}

// нормализует значения флагов и подставляет
// значения по умолчанию при невалидном вводе
func validateFlags() {
	if flagPollIntervalInSec <= 0 {
		flagPollIntervalInSec = pollIntervalInSecDefault
	}
	if flagReportIntervalInSec <= 0 {
		flagReportIntervalInSec = reportIntervalInSecDefault
	}
}
