package bfs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/headway/state"
	"github.com/itchio/lake/tlc"
	"github.com/pkg/errors"
)

type SaveAngelsParams struct {
	Consumer *state.Consumer
	Folder   string
	Receipt  *Receipt
}

type SaveAngelsFunc func() error

type SaveAngelsResult struct {
	Files []string
}

/**
 * An angel redemption is performed when we need to run arbitrary installers
 * that do not report which files they wrote.
 *
 * Conceptually:
 *   - We rename the existing folder to a temporary folder
 *   - We install to a fresh folder
 *   - We merge angels with the fresh folder
 *   - We clean up the temporary folder
 *
 * Angels are files that have been written by the game (like configurations
 * or save files) or the user (by applying a mod manually, for example)
 * They're not part of a fresh installation of the previous version,
 * but we do want to keep them around.
 *
 * See also: bust ghosts.
 */
func SaveAngels(params *SaveAngelsParams, innerTask SaveAngelsFunc) (*SaveAngelsResult, error) {
	destPath := params.Folder
	consumer := params.Consumer

	receipt := params.Receipt

	switching := true
	merging := true

	if !receipt.HasFiles() {
		consumer.Infof("No receipt found, won't save any angels")
		merging = false
	}

	if !Exists(params.Folder) {
		consumer.Infof("Destination doesn't exist yet, will not perform a switcheroo")
		switching = false
	}

	previousPath := destPath + "-previous"
	if switching {
		err := os.Rename(destPath, previousPath)
		if err != nil {
			return nil, errors.Wrap(err, "renaming destination to temporary name")
		}
	}

	err := Mkdir(destPath)
	if err != nil {
		return nil, errors.Wrap(err, "creating fresh destination folder")
	}

	innerErr := innerTask()
	if innerErr != nil {
		// let's just wipe the folder
		// TODO: retry logic?
		consumer.Infof("%s: wiping because inner task failed", destPath)
		err := os.RemoveAll(destPath)
		if err != nil {
			consumer.Warnf("Could not wipe after failed inner task: %v", err)
		}

		if switching {
			// let's restore the previous folder
			consumer.Infof("%s: restoring", previousPath)
			err := os.Rename(previousPath, destPath)
			if err != nil {
				consumer.Warnf("Could not restore previous folder after inner task: %v", err)
			}
		}

		return nil, errors.Wrap(innerErr, "performing inner task while saving angels")
	}

	// walk the freshly-installed dir now so we can store
	// it in the receipt later. we walk it before saving
	// angels so they don't end up in the receipt
	newContainer, err := Walk(destPath)
	if err != nil {
		// if we can't walk it, we can't write a proper receipt,
		// and we're kinda out of options
		return nil, errors.Wrap(err, "walking destination path to determine new files")
	}

	newPaths := ContainerPaths(newContainer)

	if merging {
		redempt := func() error {
			// now, save angels if any
			var previousContainer *tlc.Container
			previousContainer, err = Walk(previousPath)
			if err != nil {
				return errors.Wrap(err, "walking previous destination to determine old files")
			}

			previousPaths := ContainerPaths(previousContainer)
			// angels will contain files that were on disk but not
			// in receipt, meaning they were created by manual modding
			// or at runtime by the game/program and should be saved
			angels := Difference(receipt.Files, previousPaths)

			if len(angels) > 0 {
				examples := []string{}
				for _, angel := range SliceToLength(angels, 4) {
					examples = append(examples, filepath.Base(angel))
				}
				consumer.Infof("Saving %d angels like: %s", len(angels), strings.Join(examples, " ::: "))

				performAngelRedemption(params, previousPath, angels)
			} else {
				consumer.Infof("No angels to save")
			}
			return nil
		}

		redemptErr := redempt()
		if redemptErr != nil {
			consumer.Warnf("Error while performing redemption: %v", redemptErr)
		}
	}

	// and get rid of previous folder
	err = os.RemoveAll(previousPath)
	if err != nil {
		consumer.Warnf("could not remove temp folder %s: %v", previousPath, err)
	}

	return &SaveAngelsResult{
		Files: newPaths,
	}, nil
}

func performAngelRedemption(params *SaveAngelsParams, previousPath string, angels []string) {
	consumer := params.Consumer
	dt := NewDirTree(params.Folder)

	save := func(angel string) error {
		dark := filepath.Join(previousPath, angel)
		light := filepath.Join(params.Folder, angel)

		err := dt.EnsureParents(angel)
		if err != nil {
			return err
		}

		return os.Rename(dark, light)
	}

	for _, angel := range angels {
		err := save(angel)
		if err != nil {
			consumer.Warnf("Could not save angel %s: %v", angel, err)
			continue
		}
	}
}
