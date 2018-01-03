package inno

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/installer"
)

func (m *Manager) Uninstall(params *installer.UninstallParams) error {
	consumer := params.Consumer
	folder := params.InstallFolderPath

	consumer.Infof("%s: probing with configurator", folder)

	verdict, err := configurator.Configure(folder, false)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var chosen *configurator.Candidate
	for _, c := range verdict.Candidates {
		if c.Flavor != configurator.FlavorNativeWindows {
			consumer.Infof("%s: ignoring (not native windows)", c.Path)
			continue
		}

		if c.WindowsInfo == nil {
			consumer.Infof("%s: ignoring (nil windows info)", c.Path)
			continue
		}

		if c.WindowsInfo.InstallerType != "inno" {
			consumer.Infof("%s: ignoring (wrong installer type '%s')", c.Path, c.WindowsInfo.InstallerType)
			continue
		}

		consumer.Infof("%s: is our chosen uninstaller", c.Path)
		chosen = c
		break
	}

	if chosen == nil {
		return errors.New("could not find inno uninstaller in folder")
	}

	uninstallerPath := filepath.Join(folder, chosen.Path)
	cmdTokens := []string{
		uninstallerPath,
		"/VERYSILENT", // be vewwy vewwy quiet
	}

	consumer.Infof("launching inno uninstaller")

	// N.B: InnoSetup uninstallers are smart enough to elevate themselves.
	exitCode, err := installer.RunCommand(consumer, cmdTokens)
	err = installer.CheckExitCode(exitCode, err)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
