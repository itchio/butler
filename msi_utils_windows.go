package main

import (
	"github.com/winlabs/gowin32"
)

func installStateToString(state gowin32.InstallState) string {
	switch state {
	case gowin32.InstallStateBadConfig:
		return "Bad Config"
	case gowin32.InstallStateIncomplete:
		return "Incomplete"
	case gowin32.InstallStateSourceAbsent:
		return "Source Absent"
	case gowin32.InstallStateMoreData:
		return "More Data"
	case gowin32.InstallStateInvalidArg:
		return "Invalid Arg"
	case gowin32.InstallStateUnknown:
		return "Unknown"
	case gowin32.InstallStateBroken:
		return "Broken"
	case gowin32.InstallStateAdvertised:
		return "Advertised"
	case gowin32.InstallStateAbsent:
		return "Absent"
	case gowin32.InstallStateLocal:
		return "Local"
	case gowin32.InstallStateSource:
		return "Source"
	case gowin32.InstallStateDefault:
		return "Default"
	}
	return "<Unsupported>"
}

var msiInitialized = false

func initMsi() {
	if msiInitialized {
		return
	}

	gowin32.SetInstallerInternalUI(gowin32.InstallUILevelNone)

	msiInitialized = true
}
