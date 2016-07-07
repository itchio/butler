package gzip

import (
	"compress/gzip"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
)

type gzipCompressor struct{}

func (gc *gzipCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	writer, err := gzip.NewWriterLevel(writer, int(quality))
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return writer, nil
}

func init() {
	pwr.RegisterCompressor(pwr.CompressionAlgorithm_GZIP, &gzipCompressor{})
}
