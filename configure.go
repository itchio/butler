package main

import "github.com/itchio/butler/configurator"

func configure(root string, showSpell bool) {
	must(configurator.Configure(root, showSpell, filterPaths))
}
