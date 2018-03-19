package singlefilesink

import (
	"errors"
	"io"

	"github.com/itchio/savior"
)

type singleFileSink struct {
	pr    io.Reader
	pw    io.WriteCloser
	taken bool
}

type Sink interface {
	savior.Sink
	io.Reader
}

var _ Sink = (*singleFileSink)(nil)

func New() Sink {
	pr, pw := io.Pipe()
	return &singleFileSink{
		pr: pr,
		pw: pw,
	}
}

func (sfs *singleFileSink) Read(buf []byte) (int, error) {
	return sfs.pr.Read(buf)
}

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
