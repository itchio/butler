package main

import (
	"github.com/itchio/butler/cmd/launch/launchers/html"
	"github.com/itchio/butler/cmd/launch/launchers/native"
	"github.com/itchio/butler/cmd/launch/launchers/shell"
	"github.com/itchio/butler/cmd/launch/launchers/url"
)

func init() {
	native.Register()
	shell.Register()
	html.Register()
	url.Register()
}
