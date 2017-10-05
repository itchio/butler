package exeprops

import (
	"debug/pe"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
)

var args = struct {
	path *string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("exeprops", "(Advanced) Gives information about an .exe file").Hidden()
	args.path = cmd.Arg("path", "The exe to analyze").Required().String()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	ctx.Must(Do(*args.path))
}

func Do(path string) error {
	f, err := pe.Open(path)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer f.Close()

	props := &mansion.ExePropsResult{}

	switch f.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		props.Arch = "386"
	case pe.IMAGE_FILE_MACHINE_AMD64:
		props.Arch = "amd64"
	}

	comm.Result(props)

	return nil
}
