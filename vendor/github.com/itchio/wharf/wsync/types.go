package wsync

import (
	"hash"
	"io"
)

// Internal constant used in rolling checksum.
const _M = 1 << 16

// An OpType describes the type of a sync operation
type OpType byte

const (
	// OpBlockRange is a type of operation where a block of bytes is copied
	// from an old file into the file we're reconstructing
	OpBlockRange OpType = iota

	// OpData is a type of operation where fresh bytes are pasted into
	// the file we're reconstructing, because we weren't able to re-use
	// data from the old files set
	OpData
)

// Operation describes a step required to mutate target to align to source.
type Operation struct {
	Type       OpType
	FileIndex  int64
	BlockIndex int64
	BlockSpan  int64
	Data       []byte
}

// An OperationWriter consumes sync operations and does whatever it wants with them
type OperationWriter func(op Operation) error

// BlockHash is a signature hash item generated from target.
type BlockHash struct {
	FileIndex  int64
	BlockIndex int64
	WeakHash   uint32

	// ShortSize specifies the block size when non-zero
	ShortSize int32

	StrongHash []byte
}

// A SignatureWriter consumes block hashes and does whatever it wants with them
type SignatureWriter func(hash BlockHash) error

// Context holds the state during a sync operation
type Context struct {
	blockSize    int
	buffer       []byte
	uniqueHasher hash.Hash
}

// A Pool gives read+seek access to an ordered list of files, by index
type Pool interface {
	GetSize(fileIndex int64) int64
	GetReader(fileIndex int64) (io.Reader, error)
	GetReadSeeker(fileIndex int64) (io.ReadSeeker, error)
	Close() error
}

// A WritablePool adds writing access to the Pool type
type WritablePool interface {
	Pool

	GetWriter(fileIndex int64) (io.WriteCloser, error)
}

// A BlockLibrary contains a collection of weak+strong block hashes, indexed
// by their weak-hashes for fast lookup.
type BlockLibrary struct {
	hashLookup map[uint32][]BlockHash
}
