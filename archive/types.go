package archive

import (
	"errors"

	"github.com/itchio/wharf/state"
)

var (
	ErrUnrecognizedArchiveType = errors.New("Unrecognized archive type")
)

// Entry refers to a file entry in an archive
type Entry struct {
	Name             string
	UncompressedSize int64
}

type ListParams struct {
	Path     string
	Consumer *state.Consumer
}

type UncompressedSizeKnownFunc func(uncompressedSize int64)

type ExtractParams struct {
	Path       string
	OutputPath string

	Consumer                *state.Consumer
	OnUncompressedSizeKnown UncompressedSizeKnownFunc
}

type Handler interface {
	Name() string
	List(params *ListParams) (ListResult, error)
	Extract(params *ExtractParams) error
}

type ListResult interface {
	FormatName() string
	Entries() []*Entry
	Handler() Handler
}
