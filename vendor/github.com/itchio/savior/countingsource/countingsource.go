package countingsource

import (
	"github.com/itchio/savior"
	"github.com/pkg/errors"
)

type CountingCallback func(offset int64)

const readByteThreshold = 128 * 1024

type countingSource struct {
	source       savior.SeekSource
	cc           CountingCallback
	numReadBytes int
}

var _ savior.SeekSource = (*countingSource)(nil)

func New(source savior.SeekSource, cc CountingCallback) savior.SeekSource {
	return &countingSource{
		source: source,
		cc:     cc,
	}
}

func (cs *countingSource) Features() savior.SourceFeatures {
	return cs.source.Features()
}

func (cs *countingSource) Resume(checkpoint *savior.SourceCheckpoint) (int64, error) {
	return cs.source.Resume(checkpoint)
}

func (cs *countingSource) Read(buf []byte) (int, error) {
	n, err := cs.source.Read(buf)
	cs.numReadBytes += n
	if cs.numReadBytes > readByteThreshold {
		cs.numReadBytes = 0
		cs.cc(cs.source.Tell())
	}
	return n, err
}

func (cs *countingSource) ReadByte() (byte, error) {
	b, err := cs.source.ReadByte()
	if err == nil {
		cs.numReadBytes++
		if cs.numReadBytes > readByteThreshold {
			cs.numReadBytes = 0
			cs.cc(cs.source.Tell())
		}
	}
	return b, err
}

func (cs *countingSource) Tell() int64 {
	return cs.source.Tell()
}

func (cs *countingSource) Size() int64 {
	return cs.source.Size()
}

func (cs *countingSource) Progress() float64 {
	return cs.source.Progress()
}

func (cs *countingSource) WantSave() {
	cs.source.WantSave()
}

func (cs *countingSource) SetSourceSaveConsumer(ssc savior.SourceSaveConsumer) {
	cs.source.SetSourceSaveConsumer(ssc)
}

func (cs *countingSource) Section(start int64, size int64) (savior.SeekSource, error) {
	ss, err := cs.source.Section(start, size)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// this avoids the count jumping back
	cc := func(count int64) {
		cs.cc(start + count)
	}

	return New(ss, cc), nil
}
