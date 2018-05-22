package main

import (
	"fmt"
	"log"
	"os"

	"github.com/itchio/wizardry/wizardry/wizparser"

	"github.com/itchio/wizardry/wizardry/wizinterpreter"

	"github.com/itchio/wizardry/wizardry/wizutil"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s TARGET\n", os.Args[0])
		os.Exit(1)
	}

	target := os.Args[1]

	r, err := os.Open(target)
	if err != nil {
		panic(err)
	}

	stats, err := r.Stat()
	if err != nil {
		panic(err)
	}

	sr := wizutil.NewSliceReader(r, 0, stats.Size())

	book := make(wizparser.Spellbook)
	pc := &wizparser.ParseContext{
		Logf: log.Printf,
	}
	err = pc.ParseAll("Magdir", book)
	if err != nil {
		panic(err)
	}

	ic := &wizinterpreter.InterpretContext{
		Logf: log.Printf,
		Book: book,
	}

	res, err := ic.Identify(sr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s: %s\n", target, wizutil.MergeStrings(res))
}
