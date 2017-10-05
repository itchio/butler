package which

import (
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/kardianos/osext"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("which", "Prints the path to this binary")
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	p, err := osext.Executable()
	ctx.Must(err)

	comm.Logf("You're running butler %s, from the following path:", ctx.VersionString)
	comm.Logf("%s", p)
}
