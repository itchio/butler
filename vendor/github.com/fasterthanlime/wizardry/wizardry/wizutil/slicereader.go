package wizutil

import "io"

type SliceReader struct {
	reader io.ReaderAt
	offset int64
	size   int64
}

var _ io.ReaderAt = (*SliceReader)(nil)

func NewSliceReader(reader io.ReaderAt, offset int64, size int64) *SliceReader {
	return &SliceReader{
		reader: reader,
		offset: offset,
		size:   size,
	}
}

func (sr *SliceReader) Slice(offset int64) *SliceReader {
	return &SliceReader{
		reader: sr.reader,
		offset: sr.offset + offset,
		size:   sr.size - offset,
	}
}

func (sr *SliceReader) Cap(size int64) *SliceReader {
	return &SliceReader{
		reader: sr.reader,
		offset: sr.offset,
		size:   min(sr.size, size),
	}
}

func (sr *SliceReader) AbsoluteOffset() int64 {
	offset := sr.offset
	r := sr.reader
	for {
		if sub, ok := r.(*SliceReader); ok {
			offset += sub.offset
			r = sub
		} else {
			break
		}
	}
	return offset
}

func (sr *SliceReader) AbsoluteSize() int64 {
	size := sr.size
	r := sr.reader
	for {
		if sub, ok := r.(*SliceReader); ok {
			size = sub.size
			r = sub
		} else {
			break
		}
	}
	return size
}

func (sr *SliceReader) Size() int64 {
	return sr.size
}

func (sr *SliceReader) ReadAt(buf []byte, index int64) (int, error) {
	return sr.reader.ReadAt(buf, index+sr.offset)
}
