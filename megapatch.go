package main

import (
	"encoding/binary"
	"os"

	"gopkg.in/kothar/brotli-go.v0/dec"
)

const (
	MP_MAGIC      = uint64(0xFEF5F04A)
	MP_NUM_BLOCKS = iota
	MP_FILES
	MP_DIRS
	MP_SYMLINKS
)

func megapatch(patch string, source string, output string) {
	_patchReader, err := os.Open(patch)
	must(err)

	patchReader := dec.NewBrotliReader(_patchReader)

	var magic uint64
	must(binary.Read(patchReader, binary.LittleEndian, &magic))
	if magic != MP_MAGIC {
		Die("wrong magic number!")
	}

	Die("megapatch: stub!")
}
