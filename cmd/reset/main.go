package main

import "log"

func main() {
	if err := Generate("."); err != nil {
		log.Fatal(err)
	}
}
