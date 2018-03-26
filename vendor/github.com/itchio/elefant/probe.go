package elefant

import (
	"debug/elf"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/eos"
)

type Arch string

const (
	Arch386   = "386"
	ArchAmd64 = "amd64"
)

type ElfInfo struct {
	Arch Arch
}

type ProbeParams struct {
	// nothing yet
}

// Probe retrieves information about an ELF file
func Probe(file eos.File, params *ProbeParams) (*ElfInfo, error) {
	ef, err := elf.NewFile(file)
	if err != nil {
		return nil, errors.Wrap(err, 0)
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
