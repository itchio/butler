package cave

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/archives"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/configurator"
)

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

func getInstallerType(target string) (InstallerType, error) {
	ext := filepath.Ext(target)
	name := filepath.Base(target)

	if typ, ok := installerForExt[ext]; ok {
		comm.Logf("%s: choosing installer '%s'", name, typ)
		return typ, nil
	}

	comm.Logf("%s: no extension match, using configurator", name)

	verdict, err := configurator.Configure(target, false)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	if len(verdict.Candidates) == 1 {
		candidate := verdict.Candidates[0]
		switch candidate.Flavor {

		case configurator.FlavorNativeWindows:
			if candidate.WindowsInfo != nil && candidate.WindowsInfo.InstallerType != "" {
				typ := (InstallerType)(candidate.WindowsInfo.InstallerType)
				comm.Logf(`%s: windows installer of type %s`, name, typ)
				return typ, nil
			} else {
				comm.Logf(`%s: native windows executable, but not an installer`, name)
				return InstallerTypeNaked, nil
			}

		case configurator.FlavorNativeMacos:
			comm.Logf(`%s: native macOS executable`, name)
			return InstallerTypeNaked, nil

		case configurator.FlavorNativeLinux:
			comm.Logf(`%s: native linux executable`, name)
			return InstallerTypeNaked, nil

		case configurator.FlavorScript:
			comm.Logf(`%s: script`, name)
			if candidate.ScriptInfo != nil && candidate.ScriptInfo.Interpreter != "" {
				comm.Logf("...with interpreter %s", candidate.ScriptInfo.Interpreter)
			}
			return InstallerTypeNaked, nil

		case configurator.FlavorScriptWindows:
			comm.Logf(`%s: windows script`, name)
			return InstallerTypeNaked, nil
		}
	}

	comm.Logf("%s: no configurator match, probing as archive", name)
	info, err := archives.GetInfo(target)
	if err == nil {
		comm.Logf("%s: is archive, %s", name, info)
		return InstallerTypeArchive, nil
	}

	return InstallerTypeUnknown, nil
}
