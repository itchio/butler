package archive

import (
	"errors"

	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var (
	ErrUnrecognizedArchiveType = errors.New("Unrecognized archive type")
)

type ListParams struct {
	File      eos.File
	StagePath string

	Consumer *state.Consumer
}

type UncompressedSizeKnownFunc func(uncompressedSize int64)

type ExtractParams struct {
	File       eos.File
	StagePath  string
	OutputPath string

	Consumer                *state.Consumer
	OnUncompressedSizeKnown UncompressedSizeKnownFunc
}

type TryOpenParams struct {
	File     eos.File
	Consumer *state.Consumer
}

type Handler interface {
	Name() string
	TryOpen(params *TryOpenParams) error
	Extract(params *ExtractParams) (*Contents, error)
}

type Contents struct {
	Entries []*Entry
}

// Entry refers to a file entry in an archive
type Entry struct {
	Name             string
	UncompressedSize int64
}
