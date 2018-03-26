package nsis

import (
	"path/filepath"

	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/installer"
	"github.com/pkg/errors"
)

func (m *Manager) Uninstall(params *installer.UninstallParams) error {
	consumer := params.Consumer
	folder := params.InstallFolderPath

	consumer.Infof("%s: probing with configurator", folder)

	verdict, err := configurator.Configure(folder, false)
	if err != nil {
		return errors.Wrap(err, "looking for nsis uninstaller")
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

		if c.WindowsInfo.InstallerType != "nsis" {
			consumer.Infof("%s: ignoring (wrong installer type '%s')", c.Path, c.WindowsInfo.InstallerType)
			continue
		}

		consumer.Infof("%s: is our chosen uninstaller", c.Path)
		chosen = c
		break
	}

	if chosen == nil {
		return errors.New("could not find nsis uninstaller in folder")
	}

	uninstallerPath := filepath.Join(folder, chosen.Path)
	cmdTokens := []string{
		uninstallerPath,
		"/S", // run the uninstaller silently
	}

	pathArgs := getSeriouslyMisdesignedNsisPathArguments("_?=", params.InstallFolderPath)
	cmdTokens = append(cmdTokens, pathArgs...)

	consumer.Infof("→ Launching nsis uninstaller")

	exitCode, err := installer.RunElevatedCommand(consumer, cmdTokens)
	err = installer.CheckExitCode(exitCode, err)
	if err != nil {
		return errors.Wrap(err, "making sure nsis uninstaller ran correctly")
	}

	return nil
}
