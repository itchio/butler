package filesource

import (
	"github.com/itchio/savior"
	"github.com/itchio/savior/seeksource"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
)

func OpenPaused(name string, opts ...option.Option) (savior.FileSource, error) {
	f, err := eos.Open(name, opts...)
	if err != nil {
		return nil, err
	}

	ss := seeksource.FromFile(f)
	fs := &fileSource{
		SeekSource: ss,
		f:          f,
	}
	return fs, nil
}

func Open(name string, opts ...option.Option) (savior.FileSource, error) {
	s, err := OpenPaused(name, opts...)
	if err != nil {
		return nil, err
	}

	_, err = s.Resume(nil)
	if err != nil {
		return nil, err
	}

	return s, nil
}

type fileSource struct {
	savior.SeekSource

	f eos.File
}

func (fs *fileSource) Close() error {
	return fs.f.Close()
}
