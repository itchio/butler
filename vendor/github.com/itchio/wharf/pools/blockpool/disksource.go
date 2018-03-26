package blockpool

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
)

// DiskSource reads blocks from disk by their hash and length. It's hard-coded
// to use shake128-32 as a hashing algorithm.
type DiskSource struct {
	BasePath       string
	BlockAddresses BlockAddressMap

	Decompressor *Decompressor

	Container *tlc.Container
}

var _ Source = (*DiskSource)(nil)

// Clone returns a copy of this disk source, suitable for fan-in
func (ds *DiskSource) Clone() Source {
	dsc := &DiskSource{
		BasePath:       ds.BasePath,
		BlockAddresses: ds.BlockAddresses,

		Container: ds.Container,
	}

	if ds.Decompressor != nil {
		dsc.Decompressor = ds.Decompressor.Clone()
	}

	return dsc
}

// Fetch reads a block from disk
func (ds *DiskSource) Fetch(loc BlockLocation, data []byte) (int, error) {
	addr := ds.BlockAddresses.Get(loc)
	if addr == "" {
		return 0, errors.WithStack(fmt.Errorf("no address for block %+v", loc))
	}
	path := filepath.Join(ds.BasePath, addr)

	fr, err := os.Open(path)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	defer fr.Close()

	if ds.Decompressor == nil {
		bytesRead, err := io.ReadFull(fr, data)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				// all good
			} else {
				return 0, errors.WithStack(err)
			}
		}

		return bytesRead, nil
	}

	return ds.Decompressor.Decompress(data, fr)
}

// GetContainer returns the tlc container this disk source is paired with
func (ds *DiskSource) GetContainer() *tlc.Container {
	return ds.Container
}
