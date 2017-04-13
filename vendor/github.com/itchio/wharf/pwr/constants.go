package pwr

import (
	"encoding/binary"

	"github.com/itchio/wharf/wsync"
)

// Endianness defines the byte order of all fixed-size integers written or read by wharf
var Endianness = binary.LittleEndian

const (
	// PatchMagic is the magic number for wharf patch files (.pwr)
	PatchMagic = int32(iota + 0xFEF5F00)

	// SignatureMagic is the magic number for wharf signature files (.pws)
	SignatureMagic

	// ManifestMagic is the magic number for wharf manifest files (.pwm)
	ManifestMagic

	// WoundsMagic is the magic number for wharf wounds file (.pww)
	WoundsMagic

	// ZipIndexMagic is the magic number for wharf zip index files (.pzi)
	ZipIndexMagic
)

// ModeMask is or'd with files being applied/created
const ModeMask = 0644

// BlockSize is the standard block size files are broken into when ran through wharf's diff
const BlockSize int64 = 64 * 1024 // 64k

func mksync() *wsync.Context {
	return wsync.NewContext(int(BlockSize))
}
