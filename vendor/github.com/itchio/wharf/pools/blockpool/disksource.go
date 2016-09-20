package blockpool

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
)

type DiskSource struct {
	BasePath       string
	BlockAddresses BlockAddressMap

	Container *tlc.Container
}

var _ Source = (*DiskSource)(nil)

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

func (ds *DiskSource) GetContainer() *tlc.Container {
	return ds.Container
}
