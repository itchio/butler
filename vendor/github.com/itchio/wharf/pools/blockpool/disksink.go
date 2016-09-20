package blockpool

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/tlc"
	"golang.org/x/crypto/sha3"
)

type DiskSink struct {
	BasePath string

	Container *tlc.Container

	hashBuf []byte
	shake   sha3.ShakeHash
}

var _ Sink = (*DiskSink)(nil)

// Concurrent calls to DiskSink.Store will resulted in corrupted hashes
func (ds *DiskSink) Store(loc BlockLocation, data []byte) error {
	if ds.hashBuf == nil {
		ds.hashBuf = make([]byte, 32)
	}

	if ds.shake == nil {
		ds.shake = sha3.NewShake128()
	}

	ds.shake.Reset()
	_, err := ds.shake.Write(data)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	_, err = io.ReadFull(ds.shake, ds.hashBuf)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	addr := fmt.Sprintf("shake128-32/%x/%d", ds.hashBuf, len(data))
	path := filepath.Join(ds.BasePath, addr)

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	err = ioutil.WriteFile(path, data, 0644)
	if err != nil {
		return errors.Wrap(err, 1)
	}

	return nil
}

func (ds *DiskSink) GetContainer() *tlc.Container {
	return ds.Container
}
