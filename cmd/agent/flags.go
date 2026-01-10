package main

import (
	"flag"
)

// содержит адрес и порт сервера для приема метрик
var flagServerAddr string

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagServerAddr, "a", "localhost:8080", "server's address and port")
	flag.Parse()
}
