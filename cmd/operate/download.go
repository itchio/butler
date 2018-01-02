package operate

import (
	"path/filepath"

	"github.com/itchio/httpkit/retrycontext"
	"github.com/itchio/savior"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/butler/cmd/operate/downloadextractor"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/installer/archive/intervalsaveconsumer"
	"github.com/itchio/wharf/eos"
)

func DownloadInstallSource(oc *OperationContext, file eos.File, destPath string) error {
	consumer := oc.Consumer()

	statePath := filepath.Join(oc.StageFolder(), "download-state.dat")
	sc := intervalsaveconsumer.New(statePath, intervalsaveconsumer.DefaultInterval, consumer, oc.ctx)

	checkpoint := &savior.ExtractorCheckpoint{}
	err := sc.Load(checkpoint)
	if err != nil {
		consumer.Warnf("could not load checkpoint, ignoring: %s", err.Error())
		checkpoint = nil
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
				return ErrCancelled
			}

			if dl.IsIntegrityError(err) {
				consumer.Warnf("had integrity errors, we have to start over")
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
