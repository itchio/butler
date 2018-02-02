package main

import (
	"github.com/itchio/butler/cmd/launch/launchers/html"
	"github.com/itchio/butler/cmd/launch/launchers/native"
	"github.com/itchio/butler/cmd/launch/launchers/shell"
)

func init() {
	native.Register()
	shell.Register()
	html.Register()
}
