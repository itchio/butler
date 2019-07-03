package manager

import (
	"github.com/itchio/butler/cmd/configure"
	"github.com/itchio/dash"
	"github.com/itchio/ox"
	"github.com/itchio/headway/state"
)

func Configure(consumer *state.Consumer, installFolder string, runtime *ox.Runtime) (*dash.Verdict, error) {
	return configure.Do(&configure.Params{
		Consumer:   consumer,
		Path:       installFolder,
		OsFilter:   runtime.OS(),
		ArchFilter: runtime.Arch(),
	})
}
