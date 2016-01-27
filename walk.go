package main

import (
	"encoding/json"
	"os"

	"github.com/itchio/wharf/tlc"
)

func walk(src string) {
	blockSize := 16 * 1024
	info, err := tlc.Walk(src, blockSize)
	if err != nil {
		Die(err.Error())
	}

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(info)
}
