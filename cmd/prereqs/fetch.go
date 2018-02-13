package prereqs

import (
	"context"
	"io/ioutil"

	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/operate/loopbackconn"
	itchio "github.com/itchio/go-itchio"

	"github.com/itchio/butler/buse"

	"github.com/go-errors/errors"
)

const RedistsBaseURL = `https://dl.itch.ovh/itch-redists`

var RedistsGame = &itchio.Game{
	ID:        222417,
	Title:     "itch-redists",
	ShortText: "Redistributables for the itch.io app",
	URL:       "https://fasterthanlime.itch.io/itch-redists",
}

type TaskStateConsumer struct {
	OnState func(state *buse.PrereqsTaskStateNotification)
}

func (pc *PrereqsContext) FetchPrereqs(tsc *TaskStateConsumer, names []string) error {
	consumer := pc.Consumer

	doPrereq := func(name string) error {
		entry, err := pc.GetEntry(name)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if entry == nil {
			consumer.Warnf("Prereq (%s) not found in registry, skipping")
			return nil
		}
		destDir := pc.GetEntryDir(name)

		library, err := pc.GetLibrary()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		upload := library.GetUpload(name)
		if upload == nil {
			consumer.Warnf("Prereq (%s) not found in library, skipping")
			return nil
		}

		ctx := context.Background()
		stagingFolder, err := ioutil.TempDir("", "prereqs-install-stage")
		if err != nil {
			return errors.Wrap(err, 0)
		}
		conn := loopbackconn.New(consumer)

		conn.OnNotification("TaskStarted", loopbackconn.NoopNotificationHandler)
		conn.OnNotification("TaskSucceeded", loopbackconn.NoopNotificationHandler)

		conn.OnNotification("Operation.Progress", func(ctx context.Context, method string, params interface{}) error {
			progress := params.(*buse.OperationProgressNotification)
			tsc.OnState(&buse.PrereqsTaskStateNotification{
				Name:     name,
				Status:   buse.PrereqStatusDownloading,
				Progress: progress.Progress,
				ETA:      progress.ETA,
				BPS:      progress.BPS,
			})
			return nil
		})

		err = operate.Start(ctx, conn, &buse.OperationStartParams{
			ID:            "install-prereqs",
			StagingFolder: stagingFolder,
			Operation:     buse.OperationInstall,

			InstallParams: &buse.InstallParams{
				Game:          RedistsGame,
				InstallFolder: destDir,
				Upload:        upload,
				Build:         nil, // just go with the latest
				Credentials:   library.GetCredentials(),
			},
		})
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
