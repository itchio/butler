package main

import (
	"bytes"
	"crypto/md5"
	"io"
	"log"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"gopkg.in/kothar/brotli-go.v0/dec"
	"gopkg.in/kothar/brotli-go.v0/enc"
)

type counterWriter struct {
	count  int64
	writer io.Writer
}

func (w *counterWriter) Write(buffer []byte) (int, error) {
	written, err := w.writer.Write(buffer)
	w.count += int64(written)
	return written, err
}

func (w *counterWriter) Close() error {
	if v, ok := w.writer.(io.Closer); ok {
		return v.Close()
	}

	return nil
}

func testBrotli() {
	start := time.Now()

	src := os.Args[2]

	stats, err := os.Lstat(src)
	if err != nil {
		panic(err)
	}

	log.Println("Uncompressed size is", stats.Size(), "bytes =", humanize.Bytes(uint64(stats.Size())))

	srcReader, err := os.Open(src)
	if err != nil {
		panic(err)
	}

	h := md5.New()
	io.Copy(h, srcReader)

	originalHash := h.Sum(nil)
	log.Printf("Uncompressed hash is %x\n", originalHash)

	start = time.Now()

	params := enc.NewBrotliParams()

	for q := 0; q <= 9; q++ {
		pr, pw := io.Pipe()
		cw := &counterWriter{writer: pw}

		var compressedSize int64

		go func() {
			params.SetQuality(q)
			writer := enc.NewBrotliWriter(params, cw)
			defer writer.Close()

			srcReader.Seek(0, os.SEEK_SET)
			_, err = io.Copy(writer, srcReader)
			if err != nil {
				panic(err)
			}
		}()

		reader := dec.NewBrotliReader(pr)
		h.Reset()

		_, err = io.Copy(h, reader)
		if err != nil {
			panic(err)
		}

		decodedHash := h.Sum(nil)

		compressedSize = cw.count
		log.Println("{,de}compressed (q=", q, ") to", humanize.Bytes(uint64(compressedSize)), "in", time.Since(start))

		if !bytes.Equal(originalHash, decodedHash) {
			log.Printf("Decoded output does not match original input: %x\n", decodedHash)
			return
		}
		start = time.Now()
	}

	log.Println("Round-trip through brotli successful!")
}
