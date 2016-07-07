package pwr

import (
	"fmt"
	"io"
	"log"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/wire"
)

// A Compressor can compress a stream given a quality setting
type Compressor interface {
	Apply(writer io.Writer, quality int32) (io.Writer, error)
}

// A Decompressor can decompress a stream with a given algorithm
type Decompressor interface {
	Apply(reader io.Reader) (io.Reader, error)
}

var compressors map[CompressionAlgorithm]Compressor
var decompressors map[CompressionAlgorithm]Decompressor

func init() {
	compressors = make(map[CompressionAlgorithm]Compressor)
	decompressors = make(map[CompressionAlgorithm]Decompressor)
}

// RegisterCompressor lets wharf know how to compress a stream for a given algorithm
func RegisterCompressor(a CompressionAlgorithm, c Compressor) {
	if compressors[a] != nil {
		log.Printf("RegisterCompressor: overwriting current compressor for %s\n", a)
	}
	compressors[a] = c
}

// RegisterDecompressor lets wharf know how to decompress a stream for a given algorithm
func RegisterDecompressor(a CompressionAlgorithm, d Decompressor) {
	if decompressors[a] != nil {
		log.Printf("RegisterCompressor: overwriting current decompressor for %s\n", a)
	}
	decompressors[a] = d
}

// ToString returns a human-readable description of given compression settings
func (cs *CompressionSettings) ToString() string {
	return fmt.Sprintf("%s-q%d", cs.Algorithm.String(), cs.Quality)
}

// CompressWire wraps a wire.WriteContext into a compressor, according to given settings,
// so that any messages written through the returned WriteContext will first be compressed.
func CompressWire(ctx *wire.WriteContext, compression *CompressionSettings) (*wire.WriteContext, error) {
	if compression == nil {
		return nil, errors.Wrap(fmt.Errorf("no compression specified"), 1)
	}

	if compression.Algorithm == CompressionAlgorithm_NONE {
		return ctx, nil
	}

	compressor := compressors[compression.Algorithm]
	if compressor == nil {
		return nil, errors.Wrap(fmt.Errorf("no compressor registered for %s", compression.Algorithm.String()), 1)
	}

	compressedWriter, err := compressor.Apply(ctx.Writer(), compression.Quality)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return wire.NewWriteContext(compressedWriter), nil
}

// DecompressWire wraps a wire.ReadContext into a decompressor, according to the given settings,
// so that any messages read through the returned ReadContext will first be decompressed.
func DecompressWire(ctx *wire.ReadContext, compression *CompressionSettings) (*wire.ReadContext, error) {
	if compression == nil {
		return nil, errors.Wrap(fmt.Errorf("no compression specified"), 1)
	}

	if compression.Algorithm == CompressionAlgorithm_NONE {
		return ctx, nil
	}

	decompressor := decompressors[compression.Algorithm]
	if decompressor == nil {
		return nil, errors.Wrap(fmt.Errorf("no compressor registered for %s", compression.Algorithm.String()), 1)
	}

	compressedReader, err := decompressor.Apply(ctx.Reader())
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return wire.NewReadContext(compressedReader), nil
}
