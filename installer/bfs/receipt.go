package bfs

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
	"gopkg.in/kothar/brotli-go.v0/enc"
)

type Receipt struct {
	Files         []string       `json:"files"`
	InstallerName string         `json:"installerName"`
	Game          *itchio.Game   `json:"game"`
	Upload        *itchio.Upload `json:"upload"`
	Build         *itchio.Build  `json:"build"`
}

func ReadReceipt(InstallFolder string) (*Receipt, error) {
	path := receiptPath(InstallFolder)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// that's ok, just return a nil receipt
			return nil, nil
		}
		return nil, errors.Wrap(err, 0)
	}
	defer f.Close()

	dec := json.NewDecoder(f)

	receipt := Receipt{}
	err = dec.Decode(&receipt)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return &receipt, nil
}

func (r *Receipt) WriteReceipt(InstallFolder string) error {
	path := receiptPath(InstallFolder)

	err := Mkdir(filepath.Dir(path))
	if err != nil {
		return errors.Wrap(err, 0)
	}

	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	gzf, err := os.Create(path + ".gz")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer gzf.Close()
	gzw := gzip.NewWriter(gzf)
	defer gzw.Close()

	brf, err := os.Create(path + ".brotli")
	if err != nil {
		return errors.Wrap(err, 0)
	}
	params := enc.NewBrotliParams()
	params.SetQuality(9)
	brw := enc.NewBrotliWriter(params, brf)
	defer brw.Close()

	mw := io.MultiWriter(f, gzw, brw)

	enc := json.NewEncoder(mw)
	err = enc.Encode(r)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}

func (r *Receipt) HasFiles() bool {
	return r != nil && len(r.Files) > 0
}

func receiptPath(InstallFolder string) string {
	return filepath.Join(InstallFolder, ".itch", "receipt.json")
}
