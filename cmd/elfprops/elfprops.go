package elfprops

import (
	"encoding/json"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/elefant"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
)

var args = struct {
	path  *string
	trace *bool
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	args.path = cmd.Arg("path", "The ELF binary to analyze").Required().String()
	args.trace = cmd.Flag("trace", "Also perform a dependency trace (will probably only work on Linux)").Bool()
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	f, err := eos.Open(*args.path)
	ctx.Must(err)
	defer f.Close()

	info, err := Do(f, comm.NewStateConsumer())
	ctx.Must(err)

	comm.ResultOrPrint(info, func() {
		js, err := json.MarshalIndent(info, "", "  ")
		if err == nil {
			comm.Logf(string(js))
		}
	})

	if *args.trace {
		root, err := elefant.Trace(info, *args.path)
		ctx.Must(err)

		comm.Logf("%s", root)
	}
}

func Do(f eos.File, consumer *state.Consumer) (*elefant.ElfInfo, error) {
	return elefant.Probe(f, &elefant.ProbeParams{
		Consumer: consumer,
	})
}
