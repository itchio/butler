package xzsource

import (
	"github.com/pkg/errors"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/butler/archive/szextractor/singlefilesink"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

type xzSource struct {
	// internal
	se       szextractor.SzExtractor
	sink     singlefilesink.Sink
	progress float64
	bytebuf  []byte
	err      error
}

var _ savior.Source = (*xzSource)(nil)

func New(file eos.File, consumer *state.Consumer) (*xzSource, error) {
	xs := &xzSource{
		bytebuf: []byte{0x00},
	}

	subConsumer := &state.Consumer{
		OnMessage: func(level string, message string) {
			consumer.OnMessage(level, message)
		},
		OnProgress: func(progress float64) {
			xs.progress = progress
		},
	}

	se, err := szextractor.New(file, subConsumer)
	if err != nil {
		return nil, errors.Wrap(err, "opening xz stream with 7-zip")
	}
	xs.se = se

	return xs, nil
}

func (xs *xzSource) Features() savior.SourceFeatures {
	return savior.SourceFeatures{
		Name:          "xz",
		ResumeSupport: savior.ResumeSupportNone,
	}
}

func (xs *xzSource) SetSourceSaveConsumer(ssc savior.SourceSaveConsumer) {
	// we don't support checkpoints
}

func (xs *xzSource) WantSave() {
	// we don't support checkpoints
}

func (xs *xzSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	if checkpoint != nil {
		return 0, errors.New(`xzsource doesn't support checkpoints`)
	}
	xs.sink = singlefilesink.New()

	go func() {
		defer xs.sink.Close()

		err := xs.do()
		if err != nil {
			xs.err = err
			return
		}
	}()

	return 0, nil
}

func (xs *xzSource) do() error {
	_, err := xs.se.Resume(nil, xs.sink)
	if err != nil {
		return errors.Wrap(err, "decompressing xz stream with 7-zip")
	}

	return nil
}

func (xs *xzSource) Progress() float64 {
	return xs.progress
}

func (xs *xzSource) Read(buf []byte) (int, error) {
	if xs.err != nil {
		return 0, xs.err
	}

	if xs.sink == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	return xs.sink.Read(buf)
}

func (xs *xzSource) ReadByte() (byte, error) {
	if xs.err != nil {
		return 0, xs.err
	}

	if xs.sink == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	n, err := xs.sink.Read(xs.bytebuf)
	if n == 0 {
		/* this happens when Read needs to save, but it swallows the error */
		/* we're not meant to surface them, but there's no way to handle a */
		/* short read from ReadByte, so we just read again */
		_, err = xs.sink.Read(xs.bytebuf)
	}

	return xs.bytebuf[0], err
}
