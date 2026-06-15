// Package main содержит зелёную фикстуру: os.Exit вне func main не должен считаться нарушением.
package main

import "os"

func main() {
	shutdown()
}

func shutdown() {
	os.Exit(1)
}
