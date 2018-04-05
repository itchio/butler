package prereqs

import (
	"context"
	"io/ioutil"
	"sync"
	"time"

	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/operate/loopbackconn"
	"github.com/itchio/butler/endpoints/install"
	"github.com/itchio/butler/progress"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"

	"github.com/itchio/butler/butlerd"

	"github.com/pkg/errors"
)

var RedistsGame = &itchio.Game{
	ID:        222417,
	Title:     "itch-redists",
	ShortText: "Redistributables for the itch.io app",
	URL:       "https://fasterthanlime.itch.io/itch-redists",
}

type TaskStateConsumer struct {
	OnState func(state *butlerd.PrereqsTaskStateNotification)
}

func (pc *PrereqsContext) FetchPrereqs(tsc *TaskStateConsumer, names []string) error {
	consumer := pc.Consumer

	doPrereq := func(name string) error {
		entry, err := pc.GetEntry(name)
		if err != nil {
			return errors.Wrapf(err, "getting info about prereq %s", name)
		}

		if entry == nil {
			consumer.Warnf("Prereq (%s) not found in registry, skipping")
			return nil
		}
		destDir := pc.GetEntryDir(name)

		library, err := pc.GetLibrary()
		if err != nil {
			return errors.Wrap(err, "opening prereqs library")
		}

		upload := library.GetUpload(name)
		if upload == nil {
			consumer.Warnf("Prereq (%s) not found in library, skipping")
			return nil
		}

		ctx := context.Background()
		stagingFolder, err := ioutil.TempDir("", "prereqs-install-stage")
		if err != nil {
			return errors.Wrap(err, "creating temporary directory for prereqs installation")
		}
		conn := loopbackconn.New(consumer)

		counter := progress.NewCounter()
		counter.Start()

		cancel := make(chan struct{})
		var once sync.Once
		defer once.Do(func() {
			close(cancel)
		})

		go func() {
			for {
				select {
				case <-time.After(500 * time.Millisecond):
					tsc.OnState(&butlerd.PrereqsTaskStateNotification{
						Name:     name,
						Status:   butlerd.PrereqStatusDownloading,
						Progress: counter.Progress(),
						ETA:      counter.ETA().Seconds(),
						BPS:      counter.BPS(),
					})
				case <-cancel:
				}
			}
		}()

		taskConsumer := &state.Consumer{
			OnProgress: func(progress float64) {
				counter.SetProgress(progress)
			},
		}

		rcc := *pc.RequestContext
		rcc.Conn = conn
		rcc.Consumer = taskConsumer

		_, err = install.InstallQueue(&rcc, &butlerd.InstallQueueParams{
			Game:   RedistsGame,
			Upload: upload,
			Build:  nil, // just go with the latest

			NoCave:        true,
			StagingFolder: stagingFolder,
			InstallFolder: destDir,
		})
		if err != nil {
			return errors.Wrapf(err, "queueing download+extract for prereq %s", name)
		}

		err = operate.InstallPerform(ctx, &rcc, &butlerd.InstallPerformParams{
			ID:            "install-prereqs",
			StagingFolder: stagingFolder,
		})
		once.Do(func() {
			close(cancel)
		})
		if err != nil {
			return errors.Wrapf(err, "downloading+extracting prereq %s", name)
		}

		tsc.OnState(&butlerd.PrereqsTaskStateNotification{
			Name:   name,
			Status: butlerd.PrereqStatusReady,
		})

		return nil
	}

	numWorkers := 2
	todo := make(chan string)
	done := make(chan error, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for name := range todo {
				err := doPrereq(name)
				if err != nil {
					done <- errors.Wrapf(err, "handling prereq %s", name)
					return
				}
			}
			done <- nil
		}()
	}

	for _, name := range names {
		select {
		case todo <- name:
			// good
		case err := <-done:
			// not good
			if err != nil {
				return err
			}
		}
	}
	close(todo)

	for i := 0; i < numWorkers; i++ {
		err := <-done
		if err != nil {
			return err
		}
	}

	return nil
}
