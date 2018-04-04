package pelican

import (
	"github.com/itchio/pelican/pe"

	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type ProbeParams struct {
	Consumer *state.Consumer
}

// Probe retrieves information about an PE file
func Probe(file eos.File, params *ProbeParams) (*PeInfo, error) {
	if params == nil {
		return nil, errors.New("params must be set")
	}
	consumer := params.Consumer

	pf, err := pe.NewFile(file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	info := &PeInfo{
		VersionProperties: make(map[string]string),
	}

	switch pf.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		info.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		info.Arch = "amd64"
	}

	imports, err := pf.ImportedLibraries()
	info.Imports = imports

	sect := pf.Section(".rsrc")
	if sect != nil {
		parseResources(consumer, info, sect)
	}

	return info, nil
}
