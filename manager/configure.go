package manager

import (
	"context"

	"github.com/itchio/butler/cmd/configure"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/ox"
)

func Configure(ctx context.Context, consumer *state.Consumer, installFolder string, runtime ox.Runtime) (*dash.Verdict, error) {
	return configure.Do(ctx, configure.Params{
		Consumer:   consumer,
		Path:       installFolder,
		OsFilter:   runtime.OS(),
		ArchFilter: runtime.Arch(),
	})
}
