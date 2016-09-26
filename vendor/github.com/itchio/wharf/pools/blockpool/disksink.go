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

// DiskSink stores blocks on disk by their hash and length. It's hard-coded to
// use shake128-32 as a hashing algorithm.
// If `BlockHashes` is set, will store block hashes there.
type DiskSink struct {
	BasePath string

	Container   *tlc.Container
	BlockHashes *BlockHashMap

	hashBuf []byte
	shake   sha3.ShakeHash
	writing bool
}

// Clone returns a copy of this disk sink, suitable for fan-out
func (ds *DiskSink) Clone() Sink {
	return &DiskSink{
		BasePath: ds.BasePath,

		Container:   ds.Container,
		BlockHashes: ds.BlockHashes,
	}
}

var _ Sink = (*DiskSink)(nil)

// Store should not be called concurrently, as it will result in corrupted hashes
func (ds *DiskSink) Store(loc BlockLocation, data []byte) error {
	if ds.writing {
		return fmt.Errorf("concurrent write to disksink is unsupported")
	}

	ds.writing = true
	defer func() {
		ds.writing = false
	}()

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

	if ds.BlockHashes != nil {
		ds.BlockHashes.Set(loc, append([]byte{}, ds.hashBuf...))
	}

	fileSize := ds.Container.Files[int(loc.FileIndex)].Size
	blockSize := ComputeBlockSize(fileSize, loc.BlockIndex)
	addr := fmt.Sprintf("shake128-32/%x/%d", ds.hashBuf, blockSize)
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

// GetContainer returns the container associated with this disk sink
func (ds *DiskSink) GetContainer() *tlc.Container {
	return ds.Container
}
