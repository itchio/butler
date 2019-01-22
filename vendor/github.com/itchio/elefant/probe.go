package elefant

import (
	"debug/elf"
	"strings"

	"github.com/itchio/elefant/version"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
	"github.com/pkg/errors"
)

type Arch string

const (
	Arch386     Arch = "386"
	ArchAmd64   Arch = "amd64"
	ArchUnknown Arch = ""
)

type ElfInfo struct {
	Arch          Arch     `json:"arch"`
	Imports       []string `json:"imports"`
	GlibcVersion  string   `json:"glibcVersion"`
	CxxAbiVersion string   `json:"cxxAbiVersion"`
}

type ProbeParams struct {
	Consumer *state.Consumer
}

// Probe retrieves information about an ELF file
func Probe(file eos.File, params *ProbeParams) (*ElfInfo, error) {
	var consumer *state.Consumer
	if params != nil {
		consumer = params.Consumer
	}

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
		consumer.Warnf("Could not get ELF imported libraries: %v", err)
	}
	res.Imports = libs

	syms, err := ef.ImportedSymbols()
	if err != nil {
		consumer.Warnf("Could not get ELF imported libraries: %v", err)
	}

	for _, sym := range syms {
		ver := sym.Version
		if strings.HasPrefix(ver, "GLIBC_") {
			ver := strings.TrimPrefix(ver, "GLIBC_")
			if version.GTOrEq(ver, res.GlibcVersion) {
				res.GlibcVersion = ver
			}
		} else if strings.HasPrefix(ver, "CXXABI_") {
			ver := strings.TrimPrefix(ver, "CXXABI_")
			if version.GTOrEq(ver, res.CxxAbiVersion) {
				res.CxxAbiVersion = ver
			}
		}
	}

	return res, nil
}
