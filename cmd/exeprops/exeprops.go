package exeprops

import (
	"encoding/json"

	"github.com/itchio/pelican"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/wharf/eos"
	"github.com/itchio/wharf/state"
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
	f, err := eos.Open(*args.path)
	ctx.Must(err)
	defer f.Close()

	props, err := Do(f, comm.NewStateConsumer())
	ctx.Must(err)

	comm.ResultOrPrint(props, func() {
		js, err := json.MarshalIndent(props, "", "  ")
		if err == nil {
			comm.Logf(string(js))
		}
	})

}

func Do(f eos.File, consumer *state.Consumer) (*pelican.PeInfo, error) {
	return pelican.Probe(f, &pelican.ProbeParams{
		Consumer: consumer,
	})
}
