package main

import (
	"github.com/itchio/butler/comm"
	"github.com/kardianos/osext"
)

func which() {
	p, err := osext.Executable()
	must(err)

	comm.Logf("You're running butler %s, from the following path:", versionString)
	comm.Logf("%s", p)
}
