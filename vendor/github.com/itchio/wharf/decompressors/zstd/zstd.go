package zstd

import (
	"github.com/itchio/savior"
	"github.com/itchio/wharf/pwr"
	"github.com/itchio/wharf/zstdsource"
)

type zstdDecompressor struct{}

func (zd *zstdDecompressor) Apply(source savior.Source) (savior.Source, error) {
	return zstdsource.New(source), nil
}

func init() {
	pwr.RegisterDecompressor(pwr.CompressionAlgorithm_ZSTD, &zstdDecompressor{})
}
