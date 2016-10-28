package gzip

import (
	"compress/gzip"
	"io"

	"github.com/itchio/wharf/pwr"
)

type gzipCompressor struct{}

func (gc *gzipCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	return gzip.NewWriterLevel(writer, int(quality))
}

func init() {
	pwr.RegisterCompressor(pwr.CompressionAlgorithm_GZIP, &gzipCompressor{})
}
