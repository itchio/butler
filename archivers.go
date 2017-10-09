package main

import (
	"github.com/itchio/butler/archive/backends/bah"
	"github.com/itchio/butler/archive/backends/xad"
)

func init() {
	bah.Register()
	xad.Register()
}
