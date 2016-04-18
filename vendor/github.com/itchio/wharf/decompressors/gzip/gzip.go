package gzip

import (
	"compress/gzip"
	"io"

	"github.com/itchio/wharf/pwr"
)

type gzipDecompressor struct{}

func (gd *gzipDecompressor) Apply(reader io.Reader) (io.Reader, error) {
	return gzip.NewReader(reader)
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_GZIP, &gzipDecompressor{})
}
