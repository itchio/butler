package main

import (
	"os"

	"github.com/itchio/wharf/pwr"
)

func apply(recipe string, target string, output string) {
	recipeReader, err := os.Open(recipe)
	must(err)

	StartProgress()
	err = pwr.ApplyRecipe(recipeReader, target, output, Progress)
	EndProgress()
	must(err)

	if *appArgs.verbose {
		Logf("Rebuilt source in %s", output)
	}
}
