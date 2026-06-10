// Package main содержит красную фикстуру: прямой os.Exit в func main должен быть найден анализатором.
package main

import "os"

func main() {
	os.Exit(1) // want "os.Exit call in main function is prohibited"
}
