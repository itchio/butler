package main

import (
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/configurator"
)

func configure(root string, showSpell bool, osFilter string, archFilter string) {
	startTime := time.Now()

	comm.Opf("Collecting initial candidates")

	verdict, err := configurator.Configure(root, showSpell, filterPaths)
	must(err)

	comm.Statf("Initial candidates are:\n%s", verdict)

	err = verdict.FilterPlatform(osFilter, archFilter)
	must(err)

	comm.Opf("Filtering for os %s, arch %s", osFilter, archFilter)

	comm.Statf("After platform filter, candidates are:\n%s", verdict)

	comm.Statf("Configured in %s", time.Since(startTime))

	if *appArgs.json {
		comm.Result(verdict)
	}
}
