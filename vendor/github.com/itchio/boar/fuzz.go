//+build gofuzz

package boar

import (
	"fmt"
	"io"

	"github.com/itchio/boar/memfs"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/state"
)

var _dummyConsumer = &state.Consumer{
	OnMessage: func(lvl string, message string) {
		fmt.Printf("[%s] %s", lvl, message)
	},
}

func Fuzz(data []byte) int {
	file := memfs.New(data, "data")
	params := &ProbeParams{
		File:     file,
		Consumer: _dummyConsumer,
	}

	info, err := Probe(params)
	if err != nil {
		return 0
	}

	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return 0
	}

	ex, err := info.GetExtractor(file, _dummyConsumer)
	if err != nil {
		return 0
	}

	_, err = ex.Resume(nil, &savior.NopSink{})
	if err != nil {
		return 0
	}

	return 1
}
