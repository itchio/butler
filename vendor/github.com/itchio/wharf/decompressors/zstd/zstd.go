package zstd

import (
	"io"

	"github.com/Datadog/zstd"
	"github.com/itchio/wharf/pwr"
)

type zstdDecompressor struct{}

func (zd *zstdDecompressor) Apply(reader io.Reader) (io.Reader, error) {
	return zstd.NewReader(reader), nil
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_ZSTD, &zstdDecompressor{})
}
