package manager

import (
	"github.com/itchio/butler/cmd/configure"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/ox"
)

func Configure(consumer *state.Consumer, installFolder string, runtime *ox.Runtime) (*dash.Verdict, error) {
	return configure.Do(configure.Params{
		Consumer:   consumer,
		Path:       installFolder,
		OsFilter:   runtime.OS(),
		ArchFilter: runtime.Arch(),
	})
}
