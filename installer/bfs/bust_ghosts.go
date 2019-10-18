package bfs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/itchio/headway/state"
	"github.com/itchio/screw"
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
func BustGhosts(params BustGhostsParams) error {
	if !params.Receipt.HasFiles() {
		// if we didn't have a receipt, we can't know for sure
		// which files are ghosts, so we just don't wipe anything
		params.Consumer.Infof("No receipt found, leaving potential ghosts alone")
		return nil
	}

	oldFiles := params.Receipt.Files

	ghostFiles := Difference(params.NewFiles, oldFiles)

	if screw.IsCaseInsensitiveFS() {
		// naming skills 10/10
		ghostFiles = filterWrongParentCaseGhosts(params, ghostFiles)
	}

	if len(ghostFiles) == 0 {
		params.Consumer.Infof("No ghosts there!")
		return nil
	}
	params.Consumer.Infof("Found %d ghosts", len(ghostFiles))

	if debugGhostBusting {
		params.Consumer.Debugf("== old files")
		for _, f := range oldFiles {
			params.Consumer.Debugf("  %s", f)
		}
		params.Consumer.Debugf("== new files")
		for _, f := range params.NewFiles {
			params.Consumer.Debugf("  %s", f)
		}
		params.Consumer.Debugf("== ghosts")
		for _, f := range ghostFiles {
			params.Consumer.Debugf("  %s", f)
		}
		params.Consumer.Debugf("=====================")
	}

	removeFoundGhosts(params, ghostFiles)
	return nil
}

func filterWrongParentCaseGhosts(params BustGhostsParams, ghostFiles []string) []string {
	sort.Slice(ghostFiles, func(i, j int) bool {
		return len(ghostFiles[i]) < len(ghostFiles[j])
	})
	params.Consumer.Debugf("Filtering %d ghosts for wrong case", len(ghostFiles))

	var ghostFilesOut []string

	for _, ghostFile := range ghostFiles {
		tokens := strings.Split(ghostFile, "/")
		qualifies := false

		for i := 1; i <= len(tokens); i++ {
			subtokens := tokens[:i]
			partialPath := strings.Join(subtokens, "/")
			absolutePartialPath := filepath.Join(params.Folder, partialPath)
			if screw.IsWrongCase(absolutePartialPath) {
				if debugGhostBusting {
					params.Consumer.Debugf("(%s) does not exist, disqualifying ghost (%s)", partialPath, ghostFile)
				}
				qualifies = false
				break
			}
		}

		if qualifies {
			ghostFilesOut = append(ghostFilesOut, ghostFile)
		}
	}
	return ghostFilesOut
}

func removeFoundGhosts(params BustGhostsParams, ghostFiles []string) {
	for _, ghostFile := range ghostFiles {
		absolutePath := filepath.Join(params.Folder, ghostFile)

		err := screw.Remove(absolutePath)
		if err != nil {
			params.Consumer.Debugf("Leaving ghost file behind (%s): %s", absolutePath, err.Error())
		}
	}

	dt := NewDirTree(params.Folder)
	dt.CommitFiles(ghostFiles)
	for _, ghostDir := range dt.ListRelativeDirs() {
		if ghostDir == "." {
			continue
		}

		absolutePath := filepath.Join(params.Folder, ghostDir)

		// instead of doing readdir first, we just call Remove - if it fails, the
		// directory wasn't empty, which is fine.
		// the way DirTree works, all directories that could possibly be empty
		// now (due to removal of some files) will be listed. However, some of
		// them might still contain files.
		_ = screw.Remove(absolutePath)
	}
}
