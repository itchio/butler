package bfs

import (
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
)

type BustGhostsParams struct {
	Consumer *state.Consumer
	Folder   string
	NewFiles []string
}

func BustGhosts(params *BustGhostsParams) error {
	destPath := params.Folder

	receipt, err := ReadReceipt(destPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if !receipt.HasFiles() {
		// if we didn't have a receipt, we can't know for sure
		// which files are ghosts, so we just don't wipe anything
		params.Consumer.Infof("No receipt found, leaving potential ghosts alone")
		return nil
	}

	oldFiles := receipt.Files

	ghostFiles := Difference(oldFiles, params.NewFiles)

	if len(ghostFiles) == 0 {
		params.Consumer.Infof("No ghosts there!")
		return nil
	}

	removeFoundGhosts(params, ghostFiles)
	return nil
}

func removeFoundGhosts(params *BustGhostsParams, ghostFiles []string) {
	for _, ghostFile := range ghostFiles {
		absolutePath := filepath.Join(params.Folder, ghostFile)

		err := os.Remove(absolutePath)
		if err != nil {
			params.Consumer.Infof("Could not bust ghost file %s", err.Error())
		}
	}

	dt := NewDirTree(params.Folder)
	dt.CommitFiles(ghostFiles)
	for _, ghostDir := range dt.ListRelativeDirs() {
		absolutePath := filepath.Join(params.Folder, ghostDir)

		err := os.Remove(absolutePath)
		if err != nil {
			params.Consumer.Infof("Could not bust ghost dir %s", err.Error())
		}
	}
}
