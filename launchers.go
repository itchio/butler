package main

import (
	"github.com/itchio/butler/endpoints/launch/launchers/html"
	"github.com/itchio/butler/endpoints/launch/launchers/native"
	"github.com/itchio/butler/endpoints/launch/launchers/shell"
	"github.com/itchio/butler/endpoints/launch/launchers/url"
)

func init() {
	native.Register()
	shell.Register()
	html.Register()
	url.Register()
}
