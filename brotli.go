package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/kothar/brotli-go/dec"
	"github.com/kothar/brotli-go/enc"
)

func testBrotli() {
	start := time.Now()

	src := os.Args[2]
	inputBuffer, err := ioutil.ReadFile(src)
	if err != nil {
		panic(err)
	}

	log.Println("Read file in", time.Since(start))
	log.Println("Uncompressed size is", humanize.Bytes(uint64(len(inputBuffer))))
	start = time.Now()

	var decoded []byte

	for q := 0; q <= 9; q++ {
		params := enc.NewBrotliParams()
		params.SetQuality(q)

		encoded, err := enc.CompressBuffer(params, inputBuffer, make([]byte, 1))
		if err != nil {
			panic(err)
		}

		log.Println("Compressed (q=", q, ") to", humanize.Bytes(uint64(len(encoded))), "in", time.Since(start))
		start = time.Now()

		decoded, err = dec.DecompressBuffer(encoded, make([]byte, 1))
		if err != nil {
			panic(err)
		}

		log.Println("Decompressed in", time.Since(start))
		start = time.Now()
	}

	if !bytes.Equal(inputBuffer, decoded) {
		log.Println("Decoded output does not match original input")
		return
	}

	log.Println("Compared in", time.Since(start))
	start = time.Now()

	log.Println("Round-trip through brotli successful!")
}
