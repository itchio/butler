package prereqs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/progress"
	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/redist"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

const RedistsBaseURL = `https://dl.itch.ovh/itch-redists`

type TaskStateConsumer struct {
	OnState func(state *buse.PrereqsTaskStateNotification)
}

func FetchPrereqs(consumer *state.Consumer, tsc *TaskStateConsumer, folder string, redistRegistry *redist.RedistRegistry, names []string) error {
	doPrereq := func(name string) error {
		entry := redistRegistry.Entries[name]
		if entry == nil {
			consumer.Warnf("Prereq (%s) not found in registry, skipping")
			return nil
		}

		tsc.OnState(&buse.PrereqsTaskStateNotification{
			Name:   name,
			Status: buse.PrereqStatusDownloading,
		})

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

		counter := progress.NewCounter()
		counter.Start()

		cancel := make(chan struct{})
		defer close(cancel)

		go func() {
			for {
				select {
				case <-time.After(1 * time.Second):
					tsc.OnState(&buse.PrereqsTaskStateNotification{
						Name:     name,
						Status:   buse.PrereqStatusDownloading,
						Progress: counter.Progress(),
						ETA:      counter.ETA().Seconds(),
						BPS:      counter.BPS(),
					})
				case <-cancel:
					return
				}
			}
		}()

		extractor.SetConsumer(&state.Consumer{
			OnProgress: func(progress float64) {
				counter.SetProgress(progress)
			},
		})

		_, err = extractor.Resume(nil, sink)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		tsc.OnState(&buse.PrereqsTaskStateNotification{
			Name:   name,
			Status: buse.PrereqStatusReady,
		})

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
