package bfs

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
)

type Receipt struct {
	Game   *itchio.Game   `json:"game"`
	Upload *itchio.Upload `json:"upload"`
	Build  *itchio.Build  `json:"build"`

	Files         []string `json:"files"`
	InstallerName string   `json:"installerName"`

	// optional, installer-specific fields
	MSIProductCode string `json:"msiProductCode,omitempty"`
}

func ReadReceipt(InstallFolder string) (*Receipt, error) {
	// TODO: read legacy receipts & convert them to new format

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

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	dec := json.NewDecoder(gzr)

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
	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	enc := json.NewEncoder(gzw)
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
	return filepath.Join(InstallFolder, ".itch", "receipt.json.gz")
}
