package blockpool

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
)

// DiskSource reads blocks from disk by their hash and length. It's hard-coded
// to use shake128-32 as a hashing algorithm.
type DiskSource struct {
	BasePath       string
	BlockAddresses BlockAddressMap

	Container *tlc.Container
}

var _ Source = (*DiskSource)(nil)

// Clone returns a copy of this disk source, suitable for fan-in
func (ds *DiskSource) Clone() Source {
	return &DiskSource{
		BasePath:       ds.BasePath,
		BlockAddresses: ds.BlockAddresses,

		Container: ds.Container,
	}
}

// Fetch reads a block from disk
func (ds *DiskSource) Fetch(loc BlockLocation) ([]byte, error) {
	addr := ds.BlockAddresses.Get(loc)
	if addr == "" {
		return nil, errors.Wrap(fmt.Errorf("no address for block %+v", loc), 1)
	}
	path := filepath.Join(ds.BasePath, addr)

	fr, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	defer fr.Close()

	buf, err := ioutil.ReadAll(fr)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	return buf, nil
}

// GetContainer returns the tlc container this disk source is paired with
func (ds *DiskSource) GetContainer() *tlc.Container {
	return ds.Container
}
