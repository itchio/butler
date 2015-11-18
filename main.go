package main

import (
	"fmt"
	"os"

	"github.com/itchio/butler/bcommon"
)

var version = "head"

func main() {
	if len(os.Args) < 2 {
		bcommon.Die("Missing command")
	}
	cmd := os.Args[1]

	switch cmd {
	case "version":
		fmt.Println(fmt.Sprintf("butler version %s", version))
	case "dl":
		dl()
	default:
		bcommon.Die("Invalid command")
	}
}
