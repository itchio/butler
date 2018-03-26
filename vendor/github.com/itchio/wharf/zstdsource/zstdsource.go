package zstdsource

import (
	"io"
	"runtime"

	"github.com/Datadog/zstd"
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

type zstdSource struct {
	source savior.Source

	// internal
	zd      io.ReadCloser
	bytebuf []byte
}

var _ savior.Source = (*zstdSource)(nil)

func New(source savior.Source) savior.Source {
	zs := &zstdSource{
		source:  source,
		bytebuf: []byte{0x0},
	}
	// zstd is as cgo library, if we don't call `Close` on the
	// reader we *will* leak memory. Besides, sources don't
	// implement `io.Closer` so there's no way for the consumer
	// to clean up explicitly. Instead, we set up a finalizer
	// so that it is eventually freed.
	runtime.SetFinalizer(zs, finalizer)
	return zs
}

func (zs *zstdSource) Features() savior.SourceFeatures {
	return savior.SourceFeatures{
		Name:          "zsrd",
		ResumeSupport: savior.ResumeSupportNone,
	}
}

func finalizer(zs *zstdSource) {
	if zs.zd != nil {
		zs.zd.Close()
		zs.zd = nil
	}
}

func (zs *zstdSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	if checkpoint != nil {
		return 0, errors.New("zstdsource: checkpoints not supported")
	}

	offset, err := zs.source.Resume(nil)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	if offset != 0 {
		return 0, errors.Errorf("expected underlying source to resume at 0, but got %d", offset)
	}

	if zs.zd != nil {
		// sic. we don't really care about errors from
		// closing a previous zstd reader
		zs.zd.Close()
		zs.zd = nil
	}
	zs.zd = zstd.NewReader(zs.source)

	return 0, nil
}

func (zs *zstdSource) Read(buf []byte) (int, error) {
	if zs.zd == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	return zs.zd.Read(buf)
}

func (zs *zstdSource) ReadByte() (byte, error) {
	if zs.zd == nil {
		return 0, errors.WithStack(savior.ErrUninitializedSource)
	}

	_, err := zs.zd.Read(zs.bytebuf)
	return zs.bytebuf[0], err
}

func (zs *zstdSource) Progress() float64 {
	return zs.source.Progress()
}

func (zs *zstdSource) SetSourceSaveConsumer(ssc savior.SourceSaveConsumer) {
	// we don't do saves
}

func (zs *zstdSource) WantSave() {
	// we don't do saves
}
