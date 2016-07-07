package gzip

import (
	"compress/gzip"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
)

type gzipDecompressor struct{}

func (gd *gzipDecompressor) Apply(reader io.Reader) (io.Reader, error) {
	reader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return reader, nil
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_GZIP, &gzipDecompressor{})
}
