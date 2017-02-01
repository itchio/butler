package main

import (
	"debug/pe"

	"github.com/itchio/butler/comm"
)

func exeProps(path string) {
	must(doExeProps(path))
}

// ExeProps is the result the exeprops command gives
type ExeProps struct {
	Arch string `json:"arch"`
}

func doExeProps(path string) error {
	f, err := pe.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	props := &ExeProps{}

	switch f.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		props.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		props.Arch = "amd64"
	}

	comm.Result(props)

	return nil
}
