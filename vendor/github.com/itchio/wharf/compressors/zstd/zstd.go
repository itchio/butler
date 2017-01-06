package zstd

import (
	"io"

	"github.com/Datadog/zstd"

	"github.com/itchio/wharf/pwr"
)

type zstdCompressor struct{}

func (gc *zstdCompressor) Apply(writer io.Writer, quality int32) (io.Writer, error) {
	return zstd.NewWriterLevel(writer, int(quality)), nil
}

func init() {
	pwr.RegisterCompressor(pwr.CompressionAlgorithm_ZSTD, &zstdCompressor{})
}
