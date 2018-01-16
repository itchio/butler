package gzip

import (
	"github.com/itchio/savior"
	"github.com/itchio/savior/gzipsource"
	"github.com/itchio/wharf/pwr"
)

type gzipDecompressor struct{}

func (bc *gzipDecompressor) Apply(source savior.Source) (savior.Source, error) {
	return gzipsource.New(source), nil
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_GZIP, &gzipDecompressor{})
}
