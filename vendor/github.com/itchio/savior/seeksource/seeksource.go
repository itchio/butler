package seeksource

import (
	"bufio"
	"bytes"
	"io"

	"github.com/go-errors/errors"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
)

type seekSource struct {
	rs io.ReadSeeker

	br *bufio.Reader

	offset     int64
	totalBytes int64
}

var _ savior.Source = (*seekSource)(nil)

func FromFile(file eos.File) savior.Source {
	res := &seekSource{
		rs: file,
	}

	stats, err := file.Stat()
	if err == nil {
		res.totalBytes = stats.Size()
	}

	return res
}

func FromBytes(buf []byte) savior.Source {
	return NewWithSize(bytes.NewReader(buf), int64(len(buf)))
}

// NewWithSize returns a new source that reads from an io.ReadSeeker.
// Progress() will return meaningful values if totalBytes is non-zero
func NewWithSize(rs io.ReadSeeker, totalBytes int64) savior.Source {
	return &seekSource{
		rs:         rs,
		totalBytes: totalBytes,
	}
}

func (ss *seekSource) Save() (*savior.SourceCheckpoint, error) {
	c := &savior.SourceCheckpoint{
		Offset: ss.offset,
	}
	return c, nil
}

func (ss *seekSource) Resume(c *savior.SourceCheckpoint) (int64, error) {
	if c == nil {
		ss.offset = 0
	} else {
		ss.offset = c.Offset
	}

	newOffset, err := ss.rs.Seek(ss.offset, io.SeekStart)
	if err != nil {
		return newOffset, errors.Wrap(err, 0)
	}

	ss.br = bufio.NewReader(ss.rs)

	return ss.offset, nil
}

func (ss *seekSource) Read(buf []byte) (int, error) {
	n, err := ss.br.Read(buf)
	ss.offset += int64(n)
	return n, err
}

func (ss *seekSource) ReadByte() (byte, error) {
	b, err := ss.br.ReadByte()
	if err == nil {
		ss.offset++
	}
	return b, err
}

func (ss *seekSource) Progress() float64 {
	if ss.totalBytes > 0 {
		return float64(ss.offset) / float64(ss.totalBytes)
	}

	return 0
}
