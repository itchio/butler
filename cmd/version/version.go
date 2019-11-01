package version

import (
	"log"

	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/mansion"
)

func Register(ctx *mansion.Context) {
	cmd := ctx.App.Command("version", "Prints the current version of butler")
	ctx.Register(cmd, do)
}

func do(ctx *mansion.Context) {
	log.Println(buildinfo.VersionString)
}
