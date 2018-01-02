package main

import (
	"github.com/itchio/butler/installer/archive"
	"github.com/itchio/butler/installer/naked"
	"github.com/itchio/butler/installer/nsis"
)

func init() {
	naked.Register()
	archive.Register()
	nsis.Register()
}
