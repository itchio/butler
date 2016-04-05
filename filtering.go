package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// TODO: make this customizable
var ignoredPaths = []string{
	".git",
	".hg",
	".svn",
	".DS_Store",
	"._*",
	"Thumbs.db",
}

func filterPaths(fileInfo os.FileInfo) bool {
	log.Printf("Got %d ignored paths\n", len(ignoredPaths))

	name := fileInfo.Name()
	for _, pattern := range ignoredPaths {
		match, err := filepath.Match(pattern, name)
		if err != nil {
			panic(fmt.Sprintf("Malformed ignore pattern '%s': %s", pattern, err.Error()))
		}
		if match {
			if *appArgs.verbose {
				fmt.Printf("Ignoring '%s' because of pattern '%s'\n", fileInfo.Name(), pattern)
			}
			return false
		}
	}

	return true
}
