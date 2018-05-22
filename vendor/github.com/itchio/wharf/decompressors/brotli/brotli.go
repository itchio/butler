package brotli

import (
	"github.com/itchio/savior"
	"github.com/itchio/savior/brotlisource"
	"github.com/itchio/wharf/pwr"
)

type brotliDecompressor struct{}

func (bc *brotliDecompressor) Apply(source savior.Source) (savior.Source, error) {
	return brotlisource.New(source), nil
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_BROTLI, &brotliDecompressor{})
}
