package imag

import (
	"os"
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/cave/imag/fshelp"
)

func BustGhosts(params *InstallParams, newFiles []string) error {
	destPath := params.InstallFolderPath

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

	ghostFiles := Difference(oldFiles, newFiles)

	if len(ghostFiles) == 0 {
		params.Consumer.Infof("No ghosts there!")
		return nil
	}

	removeFoundGhosts(params, ghostFiles)
	return nil
}

func removeFoundGhosts(params *InstallParams, ghostFiles []string) {
	for _, ghostFile := range ghostFiles {
		absolutePath := filepath.Join(params.InstallFolderPath, ghostFile)

		err := os.Remove(absolutePath)
		if err != nil {
			params.Consumer.Infof("Could not bust ghost file %s", err.Error())
		}
	}

	dt := fshelp.NewDirTree(params.InstallFolderPath)
	dt.CommitFiles(ghostFiles)
	for _, ghostDir := range dt.ListRelativeDirs() {
		absolutePath := filepath.Join(params.InstallFolderPath, ghostDir)

		err := os.Remove(absolutePath)
		if err != nil {
			params.Consumer.Infof("Could not bust ghost dir %s", err.Error())
		}
	}
}
