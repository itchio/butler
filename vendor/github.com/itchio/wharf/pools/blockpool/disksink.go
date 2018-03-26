package blockpool

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/itchio/wharf/tlc"
	"github.com/pkg/errors"
	"golang.org/x/crypto/sha3"
)

// DiskSink stores blocks on disk by their hash and length. It's hard-coded to
// use shake128-32 as a hashing algorithm.
// If `BlockHashes` is set, will store block hashes there.
type DiskSink struct {
	BasePath string

	Container   *tlc.Container
	BlockHashes *BlockHashMap

	Compressor *Compressor

	hashBuf []byte
	shake   sha3.ShakeHash
	writing bool
}

// Clone returns a copy of this disk sink, suitable for fan-out
func (ds *DiskSink) Clone() Sink {
	dsc := &DiskSink{
		BasePath: ds.BasePath,

		Container:   ds.Container,
		BlockHashes: ds.BlockHashes,
	}

	if ds.Compressor != nil {
		dsc.Compressor = ds.Compressor.Clone()
	}

	return dsc
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
		return errors.WithStack(err)
	}

	_, err = io.ReadFull(ds.shake, ds.hashBuf)
	if err != nil {
		return errors.WithStack(err)
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
		return errors.WithStack(err)
	}

	// create file only if it doesn't exist yet
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			// block's already there!
			return nil
		}
		return errors.WithStack(err)
	}

	defer file.Close()

	if ds.Compressor == nil {
		_, err = io.Copy(file, bytes.NewReader(data))
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		err = ds.Compressor.Compress(file, data)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// GetContainer returns the container associated with this disk sink
func (ds *DiskSink) GetContainer() *tlc.Container {
	return ds.Container
}
