package main

import (
	"fmt"
	"os"
)

var version = "head"

type butlerError struct {
	Error string
}

type butlerDownloadStatus struct {
	Percent int
}

type butlerMessage struct {
	Message string
}

func main() {
	if len(os.Args) < 2 {
		die("Missing command")
	}
	cmd := os.Args[1]

	switch cmd {
	case "version":
		fmt.Println(fmt.Sprintf("butler version %s", version))
	case "dl":
		dl()
	case "test-ssh":
		testSSH()
	case "test-brotli":
		testBrotli()
	default:
		die("Invalid command")
	}
}
