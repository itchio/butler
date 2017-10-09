package bfs

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
)

type Receipt struct {
	Files         []string `json:"files"`
	InstallerName string   `json:"installerName"`
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

	err := Mkdir(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
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
