package archive

import (
	"github.com/itchio/savior"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type SimpleExtractParams struct {
	ArchivePath       string
	DestinationFolder string
	Consumer          *state.Consumer
}

func SimpleExtract(params *SimpleExtractParams) (*savior.ExtractorResult, error) {
	f, err := eos.Open(params.ArchivePath, option.WithConsumer(params.Consumer))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer f.Close()

	ai, err := Probe(&TryOpenParams{
		Consumer: params.Consumer,
		File:     f,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	ex, err := ai.GetExtractor(f, params.Consumer)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	sink := &savior.FolderSink{
		Directory: params.DestinationFolder,
		Consumer:  params.Consumer,
	}
	defer sink.Close()

	res, err := ex.Resume(nil, sink)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return res, nil
}
