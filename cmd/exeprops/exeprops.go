package exeprops

import (
	"encoding/json"

	"github.com/itchio/pelican"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
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
	f, err := eos.Open(*args.path, option.WithConsumer(comm.NewStateConsumer()))
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
	return pelican.Probe(f, pelican.ProbeParams{
		Consumer: consumer,
	})
}
