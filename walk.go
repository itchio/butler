package main

import (
	"encoding/json"
	"os"

	"github.com/itchio/wharf/tlc"
)

func walk(src string) {
	info, err := tlc.Walk(src, filterDirs)
	if err != nil {
		Die(err.Error())
	}

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(info)
}
