package main

import (
	"flag"
)

// содержит адрес и порт для запуска сервера
var flagRunAddr string

// обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Parse()
}
