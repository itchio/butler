package main

import (
	"github.com/itchio/butler/archive/backends/bah"
	"github.com/itchio/butler/archive/backends/szah"
)

func init() {
	bah.Register()
	szah.Register()
}
