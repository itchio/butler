package prereqs

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

type PrereqStatus string

const (
	PrereqStatusPending     PrereqStatus = "pending"
	PrereqStatusDownloading PrereqStatus = "downloading"
	PrereqStatusReady       PrereqStatus = "ready"
	PrereqStatusInstalling  PrereqStatus = "installing"
	PrereqStatusDone        PrereqStatus = "done"
)

const RedistsBaseURL = `https://dl.itch.ovh/itch-redists`

type PrereqStateConsumer struct {
	OnPrereqState func(name string, status PrereqStatus, progress float64)
}

func FetchPrereqs(consumer *state.Consumer, psc *PrereqStateConsumer, folder string, redistRegistry *redist.RedistRegistry, names []string) error {
	doPrereq := func(name string) error {
		entry := redistRegistry.Entries[name]
		if entry == nil {
			consumer.Warnf("Prereq (%s) not found in registry, skipping")
			return nil
		}

		psc.OnPrereqState(name, PrereqStatusDownloading, 0)

		baseURL := getBaseURL(name)
		// TODO: skip download if existing and SHA1+SHA256 sums match
		archiveURL := fmt.Sprintf("%s/%s.7z", baseURL, name)
		destDir := filepath.Join(folder, name)

		consumer.Infof("Extracting (%s) to (%s)", archiveURL, destDir)

		err := os.MkdirAll(destDir, 0755)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		file, err := eos.Open(archiveURL)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		extractor, err := szextractor.New(file, consumer)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		sink := &savior.FolderSink{
			Consumer:  consumer,
			Directory: destDir,
		}

		extractor.SetConsumer(&state.Consumer{
			OnProgress: func(progress float64) {
				psc.OnPrereqState(name, PrereqStatusDownloading, progress)
			},
		})

		_, err = extractor.Resume(nil, sink)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		psc.OnPrereqState(name, PrereqStatusReady, 0)

		return nil
	}

	for _, name := range names {
		err := doPrereq(name)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}

func getBaseURL(name string) string {
	return fmt.Sprintf("%s/%s", RedistsBaseURL, name)
}
