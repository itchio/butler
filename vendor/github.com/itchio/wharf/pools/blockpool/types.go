package blockpool

import "github.com/itchio/wharf/tlc"

// A Source allows obtaining the contents of a block
type Source interface {
	// Fetch retrieves a certain block, given its location. The returned buffer
	// must not be modified by the callee (ie. it must be a copy)
	Fetch(location BlockLocation, data []byte) (readBytes int, err error)

	// GetContainer retrieves the container associated with this source, which
	// contains information such as paths, sizes, modes, symlinks and dirs
	GetContainer() *tlc.Container

	// Clone should return a copy of the Source, suitable for fan-in
	Clone() Source
}

// A Sink lets one store a block
type Sink interface {
	// Store must fully read/copy/handle data before returning, as it might
	// be changed afterwards.
	Store(location BlockLocation, data []byte) error

	// GetContainer retrieves the container associated with this source, which
	// contains information such as paths, sizes, modes, symlinks and dirs
	GetContainer() *tlc.Container

	// Clone should return a copy of the Sink, suitable for fan-out
	Clone() Sink
}

// A BlockLocation determines where a block lies in a given container (at which
// offset of which file).
type BlockLocation struct {
	FileIndex  int64
	BlockIndex int64
}
