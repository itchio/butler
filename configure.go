package main

import (
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/configurator"
)

func configure(root string, showSpell bool, osFilter string, archFilter string) {
	startTime := time.Now()

	comm.Opf("Collecting initial candidates")

	verdict, err := configurator.Configure(root, showSpell)
	must(err)

	comm.Statf("Initial candidates are:\n%s", verdict)

	fixedExecs, err := verdict.FixPermissions(false /* not dry run */)
	must(err)

	if len(fixedExecs) > 0 {
		comm.Statf("Fixed permissions of %d executables:", len(fixedExecs))
		for _, fixedExec := range fixedExecs {
			comm.Logf("  - %s", fixedExec)
		}
	}

	comm.Opf("Filtering for os %s, arch %s", osFilter, archFilter)

	verdict.FilterPlatform(osFilter, archFilter)

	comm.Statf("After platform filter, candidates are:\n%s", verdict)

	comm.Statf("Configured in %s", time.Since(startTime))

	if *appArgs.json {
		comm.Result(verdict)
	}
}
