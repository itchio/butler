package main

import (
	"fmt"
	"os"

	"github.com/itchio/butler/bio"
)

var version = "head"

func main() {
	if len(os.Args) < 2 {
		bio.Die("Missing command")
	}
	cmd := os.Args[1]

	switch cmd {
	case "version":
		fmt.Printf("butler version %s\n", version)
	case "dl":
		dl()
	default:
		bio.Die("Invalid command")
	}
}
