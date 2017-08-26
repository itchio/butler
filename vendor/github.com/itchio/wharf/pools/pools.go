package pools

import (
	"strings"

	"github.com/itchio/arkive/zip"

	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/pools/fspool"
	"github.com/itchio/wharf/pools/zippool"
	"github.com/itchio/wharf/tlc"
	"github.com/itchio/wharf/wsync"
)

func New(c *tlc.Container, basePath string) (wsync.Pool, error) {
	if basePath == "/dev/null" {
		return fspool.New(c, basePath), nil
	}

	fr, err := eos.Open(basePath)
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	targetInfo, err := fr.Stat()
	if err != nil {
		return nil, errors.Wrap(err, 1)
	}

	if targetInfo.IsDir() {
		err := fr.Close()
		if err != nil {
			return nil, err
		}

		return fspool.New(c, basePath), nil
	}

	if strings.HasSuffix(strings.ToLower(basePath), ".zip") {
		zr, err := zip.NewReader(fr, targetInfo.Size())
		if err != nil {
			return nil, errors.Wrap(err, 1)
		}
		return zippool.New(c, zr), nil
	}

	// assume single-file container
	return fspool.New(c, filepath.Dir(basePath)), nil
}
