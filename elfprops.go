package main

import (
	"debug/elf"

	"github.com/itchio/butler/comm"
)

func elfProps(path string) {
	must(doElfProps(path))
}

// ElfProps is the result the exeprops command gives
type ElfProps struct {
	Arch      string   `json:"arch"`
	Libraries []string `json:"libraries"`
}

func doElfProps(path string) error {
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	props := &ElfProps{}

	switch f.Machine {
	case elf.EM_386:
		props.Arch = "386"
	case elf.EM_X86_64:
		props.Arch = "amd64"
	}

	// ignoring error on purpose
	props.Libraries, _ = f.ImportedLibraries()

	comm.Result(props)

	return nil
}
