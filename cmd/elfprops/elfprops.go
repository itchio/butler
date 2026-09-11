package elfprops

import (
	"encoding/json"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/dash"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
)

var args = struct {
	path string
}{}

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("elfprops", "(Advanced) Gives information about an ELF binary").Hidden()
	cmd.Arg("path", "The ELF binary to analyze").Required().StringVar(&args.path)
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	consumer := comm.NewStateConsumer()

	f, err := eos.Open(args.path, option.WithConsumer(consumer))
	ctx.Must(err)
	defer f.Close()

	info, err := dash.ProbeELF(f)
	ctx.Must(err)

	comm.ResultOrPrint(info, func() {
		js, err := json.MarshalIndent(info, "", "  ")
		if err == nil {
			comm.Logf("%s", string(js))
		}
	})
}
