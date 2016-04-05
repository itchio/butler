package main

import (
	"io"

	"github.com/itchio/wharf/pwr"

	"gopkg.in/kothar/brotli-go.v0/dec"
	"gopkg.in/kothar/brotli-go.v0/enc"
)

type brotliCompressor struct{}

func (bc *brotliCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	params := enc.NewBrotliParams()
	params.SetQuality(int(quality))
	bw := enc.NewBrotliWriter(params, writer)
	return bw, nil
}

type brotliDecompressor struct{}

func (bc *brotliDecompressor) Apply(reader io.Reader) (io.Reader, error) {
	br := dec.NewBrotliReader(reader)
	return br, nil
}

func init() {
	pwr.RegisterCompressor(pwr.CompressionAlgorithm_BROTLI, &brotliCompressor{})
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_BROTLI, &brotliDecompressor{})
}
