package operate

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archive"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/wharf/state"
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

func getInstallerInfo(consumer *state.Consumer, target string) (*InstallerInfo, error) {
	ext := filepath.Ext(target)
	name := filepath.Base(target)

	if typ, ok := installerForExt[ext]; ok {
		consumer.Infof("%s: choosing installer '%s'", name, typ)
		return &InstallerInfo{Type: typ}, nil
	}

	consumer.Infof("%s: no extension match, using configurator", name)

	verdict, err := configurator.Configure(target, false)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if len(verdict.Candidates) == 1 {
		typ := getInstallerTypeForCandidate(consumer, name, verdict.Candidates[0])
		if typ != InstallerTypeUnknown {
			return &InstallerInfo{Type: typ}, nil
		}
	} else {
		consumer.Infof("%s: %d candidates (expected 1), ignoring", name, len(verdict.Candidates))
	}

	consumer.Infof("%s: no configurator match, probing as archive", name)
	listResult, err := archive.List(&archive.ListParams{
		Path:     target,
		Consumer: consumer,
	})
	if err == nil {
		consumer.Infof("%s: is archive, %s", name, listResult.FormatName())
		return &InstallerInfo{
			Type:              InstallerTypeArchive,
			ArchiveListResult: listResult,
		}, nil
	}

	return &InstallerInfo{
		Type: InstallerTypeUnknown,
	}, nil
}

func getInstallerTypeForCandidate(consumer *state.Consumer, name string, candidate *configurator.Candidate) InstallerType {
	switch candidate.Flavor {

	case configurator.FlavorNativeWindows:
		if candidate.WindowsInfo != nil && candidate.WindowsInfo.InstallerType != "" {
			typ := (InstallerType)(candidate.WindowsInfo.InstallerType)
			consumer.Infof("%s: windows installer of type %s", name, typ)
			return typ
		}

		consumer.Infof("%s: native windows executable, but not an installer", name)
		return InstallerTypeNaked

	case configurator.FlavorNativeMacos:
		consumer.Infof("%s: native macOS executable", name)
		return InstallerTypeNaked

	case configurator.FlavorNativeLinux:
		consumer.Infof("%s: native linux executable", name)
		return InstallerTypeNaked

	case configurator.FlavorScript:
		consumer.Infof("%s: script", name)
		if candidate.ScriptInfo != nil && candidate.ScriptInfo.Interpreter != "" {
			consumer.Infof("...with interpreter %s", candidate.ScriptInfo.Interpreter)
		}
		return InstallerTypeNaked

	case configurator.FlavorScriptWindows:
		consumer.Infof("%s: windows script", name)
		return InstallerTypeNaked

	}

	return InstallerTypeUnknown
}
