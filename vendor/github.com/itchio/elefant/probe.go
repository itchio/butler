package elefant

import (
	"debug/elf"

	"github.com/itchio/wharf/eos"
	"github.com/pkg/errors"
)

type Arch string

const (
	Arch386   = "386"
	ArchAmd64 = "amd64"
)

type ElfInfo struct {
	Arch Arch `json:"arch"`
}

type ProbeParams struct {
	// nothing yet
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
	return res, nil
}
