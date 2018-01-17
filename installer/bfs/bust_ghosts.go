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

/**
 * A ghost busting is performed after performing an install using a method
 * that lets us know exactly what was written to disk.
 *
 * In this case, we:
 *   - Install in-place, directly into the destination
 *   - Compare the previous list of installed files with the list
 *     of files we just wrote to disk
 *   - Remove all the ghosts
 *
 * Ghosts are files that were in the previous install and aren't present
 * in the new install. Since we don't want to keep old, unnecessary files
 * (that aren't angels) around, we just remove them.
 *
 * See also: save angels.
 */
func BustGhosts(params *BustGhostsParams) error {
	if !params.Receipt.HasFiles() {
		// if we didn't have a receipt, we can't know for sure
		// which files are ghosts, so we just don't wipe anything
		params.Consumer.Infof("No receipt found, leaving potential ghosts alone")
		return nil
	}

	oldFiles := params.Receipt.Files

	ghostFiles := Difference(params.NewFiles, oldFiles)

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
			params.Consumer.Infof("Could not bust ghost file '%s': %s", absolutePath, err.Error())
		}
	}

	dt := NewDirTree(params.Folder)
	dt.CommitFiles(ghostFiles)
	for _, ghostDir := range dt.ListRelativeDirs() {
		absolutePath := filepath.Join(params.Folder, ghostDir)

		err := os.Remove(absolutePath)
		if err != nil {
			params.Consumer.Infof("Could not bust ghost dir '%s': %s", absolutePath, err.Error())
		}
	}
}
