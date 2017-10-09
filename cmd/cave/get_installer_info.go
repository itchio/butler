package cave

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/archive/uniarch"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/configurator"
)

type InstallerInfo struct {
	Type              InstallerType
	ArchiveListResult archive.ListResult
}

type InstallerType string

const (
	InstallerTypeNaked       InstallerType = "naked"
	InstallerTypeArchive                   = "archive"
	InstallerTypeDMG                       = "dmg"
	InstallerTypeInno                      = "inno"
	InstallerTypeNsis                      = "nsis"
	InstallerTypeMSI                       = "msi"
	InstallerTypeUnknown                   = "unknown"
	InstallerTypeUnsupported               = "unsupported"
)

func getInstallerType(target string) (*InstallerInfo, error) {
	ext := filepath.Ext(target)
	name := filepath.Base(target)

	if typ, ok := installerForExt[ext]; ok {
		comm.Logf("%s: choosing installer '%s'", name, typ)
		return &InstallerInfo{Type: typ}, nil
	}

	comm.Logf("%s: no extension match, using configurator", name)

	verdict, err := configurator.Configure(target, false)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if len(verdict.Candidates) == 1 {
		typ := getInstallerTypeForCandidate(name, verdict.Candidates[0])
		if typ != InstallerTypeUnknown {
			return &InstallerInfo{Type: typ}, nil
		}
	} else {
		comm.Logf("%s: %d candidates (expected 1), ignoring", name, len(verdict.Candidates))
	}

	comm.Logf("%s: no configurator match, probing as archive", name)
	listResult, err := uniarch.List(&archive.ListParams{
		Path: target,
	})
	if err == nil {
		comm.Logf("%s: is archive, %s", name, listResult.FormatName())
		return &InstallerInfo{
			Type:              InstallerTypeArchive,
			ArchiveListResult: listResult,
		}, nil
	}

	return &InstallerInfo{
		Type: InstallerTypeUnknown,
	}, nil
}

func getInstallerTypeForCandidate(name string, candidate *configurator.Candidate) InstallerType {
	switch candidate.Flavor {

	case configurator.FlavorNativeWindows:
		if candidate.WindowsInfo != nil && candidate.WindowsInfo.InstallerType != "" {
			typ := (InstallerType)(candidate.WindowsInfo.InstallerType)
			comm.Logf("%s: windows installer of type %s", name, typ)
			return typ
		}

		comm.Logf("%s: native windows executable, but not an installer", name)
		return InstallerTypeNaked

	case configurator.FlavorNativeMacos:
		comm.Logf("%s: native macOS executable", name)
		return InstallerTypeNaked

	case configurator.FlavorNativeLinux:
		comm.Logf("%s: native linux executable", name)
		return InstallerTypeNaked

	case configurator.FlavorScript:
		comm.Logf("%s: script", name)
		if candidate.ScriptInfo != nil && candidate.ScriptInfo.Interpreter != "" {
			comm.Logf("...with interpreter %s", candidate.ScriptInfo.Interpreter)
		}
		return InstallerTypeNaked

	case configurator.FlavorScriptWindows:
		comm.Logf("%s: windows script", name)
		return InstallerTypeNaked

	}

	return InstallerTypeUnknown
}
