package main

import "github.com/itchio/butler/comm"

func installPrereqs(plan string) {
	must(doInstallPrereqs(plan))
}

func doInstallPrereqs(plan string) error {
	comm.Logf("stub!")
	return nil
}
