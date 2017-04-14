package main

import "configurator"

func configure(root string, showSpell bool) {
	must(configurator.Configure(root, showSpell, filterPaths))
}
