// Package notmainpkg содержит зелёную фикстуру: os.Exit внутри func main
// в пакете, отличном от main, не является нарушением.
package notmainpkg

import "os"

func main() {
	os.Exit(1)
}
