package main

import (
	"github.com/itchio/butler/installer/archive"
	"github.com/itchio/butler/installer/naked"
)

func init() {
	naked.Register()
	archive.Register()
}
