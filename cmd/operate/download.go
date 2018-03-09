package operate

import (
	"context"
	"path/filepath"

	"github.com/itchio/httpkit/retrycontext"
	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/butler/cmd/operate/downloadextractor"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/installer/archive/intervalsaveconsumer"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

func DownloadInstallSource(consumer *state.Consumer, stageFolder string, ctx context.Context, file eos.File, destPath string) error {
	statePath := filepath.Join(stageFolder, "download-state.dat")
	sc := intervalsaveconsumer.New(statePath, intervalsaveconsumer.DefaultInterval, consumer, ctx)

	checkpoint, err := sc.Load()
	if err != nil {
		consumer.Warnf("Could not load checkpoint: %s", err.Error())
	}

	destName := filepath.Base(destPath)
	sink := &savior.FolderSink{
		Directory: filepath.Dir(destPath),
		Consumer:  consumer,
	}

	retryCtx := retrycontext.NewDefault()
	retryCtx.Settings.Consumer = comm.NewStateConsumer()

	tryDownload := func() error {
		ex := downloadextractor.New(file, destName)
		ex.SetConsumer(consumer)
		ex.SetSaveConsumer(sc)
		_, err := ex.Resume(checkpoint, sink)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	for retryCtx.ShouldTry() {
		err := tryDownload()
		if err != nil {
			if errors.Is(err, savior.ErrStop) {
				return &buse.ErrCancelled{}
			}

			if dl.IsIntegrityError(err) {
				consumer.Warnf("Had integrity errors, we have to start over")
				checkpoint = nil
				retryCtx.Retry(err.Error())
				continue
			}

			// if it's not an integrity error, just bubble it up
			return err
		}

		return nil
	}

	return errors.New("download: too many errors, giving up")
}
