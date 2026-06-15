package main

import (
	"fmt"
	"log"
)

func main() {
	log.SetFlags(0)

	generatedFiles, err := Generate(".")
	if err != nil {
		log.Fatal(err)
	}

	if len(generatedFiles) == 0 {
		fmt.Println("reset: ¯＼_(ツ)_/¯ no structures with // generate:reset found")
		return
	}

	fmt.Printf("reset: (*_*) generated %d file(s)\n", len(generatedFiles))
	for _, file := range generatedFiles {
		fmt.Printf("reset: wrote %s\n", file)
	}
}
