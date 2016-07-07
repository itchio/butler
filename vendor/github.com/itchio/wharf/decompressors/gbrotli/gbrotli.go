package gbrotli

import (
	"io"

	"github.com/dsnet/compress/brotli"
	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pwr"
)

type brotliDecompressor struct{}

func (bc *brotliDecompressor) Apply(reader io.Reader) (io.Reader, error) {
	br, err := brotli.NewReader(reader, nil)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}
	return br, nil
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_BROTLI, &brotliDecompressor{})
}
