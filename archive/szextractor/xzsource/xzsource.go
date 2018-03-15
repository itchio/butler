package xzsource

import (
	"io"

	"github.com/go-errors/errors"

	"github.com/itchio/butler/archive/szextractor"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

type xzSource struct {
	// internal
	se       szextractor.SzExtractor
	sink     *singleFileSink
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
		return nil, errors.Wrap(err, 0)
	}
	xs.se = se

	return xs, nil
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
	xs.sink = newSingleFileSink()

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
		return errors.Wrap(err, 0)
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
		return 0, errors.Wrap(savior.ErrUninitializedSource, 0)
	}

	return xs.sink.pr.Read(buf)
}

func (xs *xzSource) ReadByte() (byte, error) {
	if xs.err != nil {
		return 0, xs.err
	}

	if xs.sink == nil {
		return 0, errors.Wrap(savior.ErrUninitializedSource, 0)
	}

	n, err := xs.sink.pr.Read(xs.bytebuf)
	if n == 0 {
		/* this happens when Read needs to save, but it swallows the error */
		/* we're not meant to surface them, but there's no way to handle a */
		/* short read from ReadByte, so we just read again */
		_, err = xs.sink.pr.Read(xs.bytebuf)
	}

	return xs.bytebuf[0], err
}

//

type singleFileSink struct {
	pr    io.Reader
	pw    io.WriteCloser
	taken bool
}

func newSingleFileSink() *singleFileSink {
	pr, pw := io.Pipe()
	return &singleFileSink{
		pr: pr,
		pw: pw,
	}
}

var _ savior.Sink = (*singleFileSink)(nil)

func (sfs *singleFileSink) Mkdir(entry *savior.Entry) error {
	// we don't mkdir by design
	return nil
}

func (sfs *singleFileSink) Symlink(entry *savior.Entry, linkname string) error {
	// we don't symlink by design
	return nil
}

func (sfs *singleFileSink) GetWriter(entry *savior.Entry) (savior.EntryWriter, error) {
	if sfs.taken {
		return nil, errors.New("singleFileSink: already returned a writer")
	}
	sfs.taken = true
	return &nopSync{sfs.pw}, nil
}

func (sfs *singleFileSink) Preallocate(entry *savior.Entry) error {
	// we don't preallocate by design
	return nil
}

func (sfs *singleFileSink) Nuke() error {
	// we don't nuke by design
	return nil
}

func (sfs *singleFileSink) Close() error {
	return sfs.pw.Close()
}

//

type nopSync struct {
	io.WriteCloser
}

var _ savior.EntryWriter = (*nopSync)(nil)

func (ns *nopSync) Sync() error {
	// we don't sync by design
	return nil
}
