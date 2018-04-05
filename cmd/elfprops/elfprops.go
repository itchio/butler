package elfprops

import (
	"encoding/json"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/elefant"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/eos/option"
	"github.com/itchio/wharf/state"
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
	consumer := comm.NewStateConsumer()

	f, err := eos.Open(*args.path, option.WithConsumer(consumer))
	ctx.Must(err)
	defer f.Close()

	props, err := Do(f, consumer)
	ctx.Must(err)

	comm.ResultOrPrint(props, func() {
		js, err := json.MarshalIndent(props, "", "  ")
		if err == nil {
			comm.Logf(string(js))
		}
	})
}

func Do(f eos.File, consumer *state.Consumer) (*elefant.ElfInfo, error) {
	return elefant.Probe(f, &elefant.ProbeParams{
		// nothing so far
	})
}
