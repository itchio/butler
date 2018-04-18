package main

import (
	"flag"
	"log"
	"path/filepath"

	"github.com/itchio/boar"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"

	"net/http"
	_ "net/http/pprof"
)

var extract bool

func init() {
	flag.BoolVar(&extract, "extract", false, "Perform in-memory extraction")
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:9000", nil))
	}()

	log.SetFlags(0)
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		log.Fatal("Usage: lilboar FILE [...FILE]")
	}

	consumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			log.Printf("%s", msg)
		},
	}

	doFile := func(filePath string) {
		file, err := eos.Open(filePath)
		if err != nil {
			consumer.Errorf("%s: %v", filePath, err)
			return
		}
		defer file.Close()

		info, err := boar.Probe(&boar.ProbeParams{
			File:     file,
			Consumer: consumer,
		})
		if err != nil {
			consumer.Errorf("%s: %v", filePath, err)
			return
		}

		consumer.Infof("%s: %s", filepath.Base(filePath), info)

		if extract {
			ex, err := info.GetExtractor(file, consumer)
			if err != nil {
				consumer.Errorf("%s: %v", filePath, err)
				return
			}

			_, err = ex.Resume(nil, &savior.NopSink{})
			if err != nil {
				consumer.Errorf("%s: %v", filePath, err)
				return
			}
		}
	}

	for _, arg := range args {
		doFile(arg)
	}
}
