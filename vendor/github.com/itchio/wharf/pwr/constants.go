package pwr

import (
	"encoding/binary"

	"github.com/itchio/wharf/sync"
)

// Endianness defines the byte order of all fixed-size integers written or read by wharf
var Endianness = binary.LittleEndian

const (
	// PatchMagic is the magic number for wharf patch files (.pwr)
	PatchMagic = int32(iota + 0xFEF5F00)

	// SignatureMagic is the magic number for wharf signature files (.pws)
	SignatureMagic
)

// ModeMask is or'd with files being applied/created
const ModeMask = 0644

// BlockSize is the standard block size files are broken into when ran through wharf's diff
var BlockSize = 64 * 1024 // 64k

func mksync() *sync.Context {
	return sync.NewContext(BlockSize)
}
