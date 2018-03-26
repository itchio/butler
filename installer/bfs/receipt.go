package bfs

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"

	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

// A Receipt describes what was installed to a specific folder.
//
// It's compressed and written to `./.itch/receipt.json.gz` every
// time an install operation completes successfully, and is used
// in further install operations to make sure ghosts are busted and/or
// angels are saved.
type Receipt struct {
	// The itch.io game installed at this location
	Game *itchio.Game `json:"game"`
	// The itch.io upload installed at this location
	Upload *itchio.Upload `json:"upload"`
	// The itch.io build installed at this location. Null for non-wharf upload.
	Build *itchio.Build `json:"build"`

	// A list of installed files (slash-separated paths, relative to install folder)
	Files []string `json:"files"`
	// The installer used to install at this location
	// @optional
	InstallerName string `json:"installerName"`

	// If this was installed from an MSI package, the product code,
	// used for a clean uninstall.
	// @optional
	MSIProductCode string `json:"msiProductCode,omitempty"`
}

func ReadReceipt(InstallFolder string) (*Receipt, error) {
	path := ReceiptPath(InstallFolder)

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// that's ok, just return a nil receipt
			return nil, nil
		}
		return nil, errors.Wrap(err, "reading receipt")
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.Wrap(err, "creating decompressor for receipt")
	}

	dec := json.NewDecoder(gzr)

	receipt := Receipt{}
	err = dec.Decode(&receipt)
	if err != nil {
		return nil, errors.Wrap(err, "decoding receipt")
	}

	return &receipt, nil
}

func (r *Receipt) WriteReceipt(InstallFolder string) error {
	if _, err := os.Stat(InstallFolder); err != nil {
		// don't create the install folder if it doesn't exist
		return errors.Errorf("Refusing to write receipt to non-existent directory: %s", err.Error())
	}

	path := ReceiptPath(InstallFolder)

	err := Mkdir(filepath.Dir(path))
	if err != nil {
		return errors.Wrap(err, "creating receipt parent directory")
	}

	f, err := os.Create(path)
	if err != nil {
		return errors.Wrap(err, "creating receipt file")
	}
	defer f.Close()
	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	enc := json.NewEncoder(gzw)
	err = enc.Encode(r)
	if err != nil {
		return errors.Wrap(err, "encoding receipt")
	}

	return nil
}

func (r *Receipt) HasFiles() bool {
	return r != nil && len(r.Files) > 0
}

func ReceiptPath(InstallFolder string) string {
	return filepath.Join(InstallFolder, ".itch", "receipt.json.gz")
}
