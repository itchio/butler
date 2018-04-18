package boar

import (
	"errors"

	"github.com/itchio/dash"
	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var (
	ErrUnrecognizedArchiveType = errors.New("Unrecognized archive type")
)

type LoadFunc func(state interface{}) error
type SaveFunc func(state interface{}) error

type ExtractParams struct {
	File       eos.File
	StagePath  string
	OutputPath string

	Consumer *state.Consumer

	Load LoadFunc
	Save SaveFunc
}

type ProbeParams struct {
	File      eos.File
	Consumer  *state.Consumer
	Candidate *dash.Candidate
	OnEntries func(entries []*savior.Entry)
}

type Contents struct {
	Entries []*Entry
}

// Entry refers to a file entry in an archive
type Entry struct {
	Name             string
	UncompressedSize int64
}
