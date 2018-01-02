package bfs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/tlc"
)

type SaveAngelsParams struct {
	Consumer *state.Consumer
	Folder   string
}

type SaveAngelsFunc func() error

type SaveAngelsResult struct {
	Files []string
}

func SaveAngels(params *SaveAngelsParams, innerTask SaveAngelsFunc) (*SaveAngelsResult, error) {
	destPath := params.Folder

	receipt, err := ReadReceipt(destPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	needSwitcheroo := true

	if !receipt.HasFiles() {
		params.Consumer.Infof("No receipt found, will not perform a switcheroo")
		needSwitcheroo = false
	} else if !Exists(params.Folder) {
		params.Consumer.Infof("Destination doesn't exist yet, will not perform a switcheroo")
		needSwitcheroo = false
	}

	previousPath := destPath + "-previous"
	if needSwitcheroo {
		err := os.Rename(previousPath, destPath)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}

	err = Mkdir(destPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var previousContainer *tlc.Container
	walkResult := make(chan error)
	if needSwitcheroo {
		go func() {
			var err error
			previousContainer, err = Walk(previousPath)
			if err != nil {
				walkResult <- errors.Wrap(err, 0)
				return
			}
			walkResult <- nil
		}()
	}

	err = innerTask()
	if err != nil {
		// FIXME: uhh we don't do any cleanup if we err here?
		return nil, errors.Wrap(err, 0)
	}

	// walk the freshly-installed dir now so we can store
	// it in the receipt later
	newContainer, err := Walk(destPath)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	newPaths := ContainerPaths(newContainer)

	if needSwitcheroo {
		// now, save angels if any
		err = <-walkResult
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		previousPaths := ContainerPaths(previousContainer)
		angels := Difference(previousPaths, newPaths)

		if len(angels) > 0 {
			examples := []string{}
			for _, angel := range SliceToLength(angels, 4) {
				examples = append(examples, filepath.Base(angel))
			}
			params.Consumer.Infof("Saving %d angels like: %s", strings.Join(examples, ", "))

			err = performAngelRedemption(params, previousPath, angels)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		} else {
			params.Consumer.Infof("No angels to save")
		}
	}

	return &SaveAngelsResult{
		Files: newPaths,
	}, nil
}

func performAngelRedemption(params *SaveAngelsParams, previousPath string, angels []string) error {
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
			params.Consumer.Warnf("Could not save angel %s: %s", angel, err.Error())
			continue
		}
	}

	return nil
}
