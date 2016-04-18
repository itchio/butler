package cbrotli

import (
	"io"

	"gopkg.in/kothar/brotli-go.v0/enc"

	"github.com/itchio/wharf/pwr"
)

type brotliCompressor struct{}

func (gc *brotliCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	params := enc.NewBrotliParams()
	params.SetQuality(int(quality))
	return enc.NewBrotliWriter(params, writer), nil
}

func init() {
	pwr.RegisterCompressor(pwr.CompressionAlgorithm_BROTLI, &brotliCompressor{})
}
