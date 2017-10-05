package elfprops

import (
	"debug/elf"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	args.path = cmd.Arg("path", "The ELF binary to analyze").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.path))
}

func Do(path string) error {
	f, err := elf.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	props := &mansion.ElfPropsResult{}

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
