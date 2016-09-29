package pools

import (
	"archive/zip"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

func New(c *tlc.Container, basePath string) (wsync.Pool, error) {
	if basePath == "/dev/null" {
		return fspool.New(c, basePath), nil
	}

	targetInfo, err := os.Lstat(basePath)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	if targetInfo.IsDir() {
		return fspool.New(c, basePath), nil
	} else {
		fr, err := os.Open(basePath)
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		zr, err := zip.NewReader(fr, targetInfo.Size())
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}

		return zippool.New(c, zr), nil
	}
}
