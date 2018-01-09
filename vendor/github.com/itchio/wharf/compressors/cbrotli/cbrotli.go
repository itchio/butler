package cbrotli

import (
	"io"

	"github.com/itchio/go-brotli/enc"
	"github.com/itchio/wharf/pwr"
)

type brotliCompressor struct{}

func (bc *brotliCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	return enc.NewBrotliWriter(writer, &enc.BrotliWriterOptions{
		Quality: int(quality),
	}), nil
}

func init() {
	pwr.RegisterCompressor(pwr.CompressionAlgorithm_BROTLI, &brotliCompressor{})
}
