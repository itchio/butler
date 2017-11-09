package bfs

import (
	"os"
	"path/filepath"

	"github.com/itchio/wharf/state"
)

var debugGhostBusting = os.Getenv("BUTLER_LOUD_GHOSTS") == "1"

type BustGhostsParams struct {
	Consumer *state.Consumer
	Folder   string
	NewFiles []string
	Receipt  *Receipt
}

func BustGhosts(params *BustGhostsParams) error {
	if !params.Receipt.HasFiles() {
		// if we didn't have a receipt, we can't know for sure
		// which files are ghosts, so we just don't wipe anything
		params.Consumer.Infof("No receipt found, leaving potential ghosts alone")
		return nil
	}

	oldFiles := params.Receipt.Files

	ghostFiles := Difference(oldFiles, params.NewFiles)

	if len(ghostFiles) == 0 {
		params.Consumer.Infof("No ghosts there!")
		return nil
	}

	if debugGhostBusting {
		params.Consumer.Infof("== old files")
		for _, f := range oldFiles {
			params.Consumer.Infof("  %s", f)
		}
		params.Consumer.Infof("== new files")
		for _, f := range params.NewFiles {
			params.Consumer.Infof("  %s", f)
		}
		params.Consumer.Infof("== ghosts")
		for _, f := range ghostFiles {
			params.Consumer.Infof("  %s", f)
		}
		params.Consumer.Infof("=====================")
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
