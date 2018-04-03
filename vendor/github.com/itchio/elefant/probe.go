package elefant

import (
	"debug/elf"

	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type Arch string

const (
	Arch386   = "386"
	ArchAmd64 = "amd64"
)

type ElfInfo struct {
	Arch    Arch     `json:"arch"`
	Imports []string `json:"imports"`
}

type ProbeParams struct {
	Consumer *state.Consumer
}

// Probe retrieves information about an ELF file
func Probe(file eos.File, params *ProbeParams) (*ElfInfo, error) {
	ef, err := elf.NewFile(file)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &ElfInfo{}

	switch ef.Machine {
	case elf.EM_386:
		res.Arch = Arch386
	case elf.EM_X86_64:
		res.Arch = ArchAmd64
	}

	libs, err := ef.ImportedLibraries()
	if err != nil {
		params.Consumer.Warnf("Could not get ELF imported libraries: %v", err)
	}
	res.Imports = libs

	return res, nil
}
