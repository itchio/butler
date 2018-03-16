package manager

import (
	"github.com/itchio/butler/cmd/configure"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/wharf/state"
)

func Configure(consumer *state.Consumer, installFolder string, runtime *Runtime) (*configurator.Verdict, error) {
	return configure.Do(&configure.Params{
		Consumer:   consumer,
		Path:       installFolder,
		OsFilter:   runtime.OS(),
		ArchFilter: runtime.Arch(),
	})
}
