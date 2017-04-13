package main

import (
	"compress/flate"
	"fmt"
	"log"
	"os"

	"io"

	"io/ioutil"

	"github.com/itchio/arkive/zip"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: ./sample FILE.ZIP")
		os.Exit(1)
	}

	path := os.Args[1]

	stats, err := os.Lstat(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	zf, err := os.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}

	zr, err := zip.NewReader(zf, stats.Size())
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, f := range zr.File {
		log.Printf("- %s (method %d)\n", f.Name, f.Method)

		if f.FileInfo().IsDir() {
			continue
		}

		offset, err := f.DataOffset()
		if err != nil {
			log.Fatal(err.Error())
		}

		sr := io.NewSectionReader(zf, offset, int64(f.CompressedSize64))

		if f.Method == zip.Deflate {
			fr := flate.NewReader(sr)
			_, err = io.Copy(ioutil.Discard, fr)
			if err != nil {
				log.Printf("flate error: %s\n", err.Error())
			}
		} else if f.Method == zip.Store {
			_, err = io.Copy(ioutil.Discard, sr)
			if err != nil {
				log.Printf("store error: %s\n", err.Error())
			}
		} else {
			log.Printf("unknown method %d\n", f.Method)
		}
	}

	// flate.NewReader(bytes.NewReader([]byte{}))
}
