package main

import (
	"github.com/itchio/butler/cmd/apply"
	"github.com/itchio/butler/cmd/clean"
	"github.com/itchio/butler/cmd/configure"
	"github.com/itchio/butler/cmd/cp"
	"github.com/itchio/butler/cmd/diff"
	"github.com/itchio/butler/cmd/ditto"
	"github.com/itchio/butler/cmd/dl"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/cmd/elfprops"
	"github.com/itchio/butler/cmd/exeprops"
	"github.com/itchio/butler/cmd/fetch"
	"github.com/itchio/butler/cmd/file"
	"github.com/itchio/butler/cmd/heal"
	"github.com/itchio/butler/cmd/indexzip"
	"github.com/itchio/butler/cmd/login"
	"github.com/itchio/butler/cmd/logout"
	"github.com/itchio/butler/cmd/ls"
	"github.com/itchio/butler/cmd/mkdir"
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/pipe"
	"github.com/itchio/butler/cmd/prereqs"
	"github.com/itchio/butler/cmd/probe"
	"github.com/itchio/butler/cmd/push"
	"github.com/itchio/butler/cmd/service"
	"github.com/itchio/butler/cmd/sign"
	"github.com/itchio/butler/cmd/sizeof"
	"github.com/itchio/butler/cmd/status"
	"github.com/itchio/butler/cmd/untar"
	"github.com/itchio/butler/cmd/unzip"
	"github.com/itchio/butler/cmd/upgrade"
	"github.com/itchio/butler/cmd/verify"
	"github.com/itchio/butler/cmd/version"
	"github.com/itchio/butler/cmd/walk"
	"github.com/itchio/butler/cmd/which"
	"github.com/itchio/butler/cmd/wipe"
	"github.com/itchio/butler/mansion"
)

// Each of these specify their own arguments and flags in
// their own package.
func registerCommands(ctx *mansion.Context) {
	// documented commands

	login.Register(ctx)
	logout.Register(ctx)

	push.Register(ctx)
	fetch.Register(ctx)
	status.Register(ctx)

	file.Register(ctx)
	ls.Register(ctx)

	which.Register(ctx)
	version.Register(ctx)
	upgrade.Register(ctx)

	sign.Register(ctx)
	verify.Register(ctx)
	diff.Register(ctx)
	apply.Register(ctx)
	heal.Register(ctx)

	// hidden commands

	dl.Register(ctx)
	cp.Register(ctx)
	wipe.Register(ctx)
	sizeof.Register(ctx)
	mkdir.Register(ctx)
	ditto.Register(ctx)
	probe.Register(ctx)

	clean.Register(ctx)
	walk.Register(ctx)

	msi.Register(ctx)
	prereqs.Register(ctx)

	unzip.Register(ctx)
	untar.Register(ctx)
	indexzip.Register(ctx)

	pipe.Register(ctx)
	elevate.Register(ctx)

	exeprops.Register(ctx)
	elfprops.Register(ctx)

	configure.Register(ctx)
	operate.Register(ctx)

	service.Register(ctx)
}
