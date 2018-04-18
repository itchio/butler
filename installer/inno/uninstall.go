package inno

import (
	"path/filepath"

	"github.com/itchio/butler/filtering"

	"github.com/itchio/butler/installer"
	"github.com/itchio/dash"
	"github.com/pkg/errors"
)

func (m *Manager) Uninstall(params *installer.UninstallParams) error {
	consumer := params.Consumer
	folder := params.InstallFolderPath

	consumer.Infof("%s: probing with configurator", folder)

	verdict, err := dash.Configure(folder, &dash.ConfigureParams{
		Consumer: params.Consumer,
		Filter:   filtering.FilterPaths,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	var chosen *dash.Candidate
	for _, c := range verdict.Candidates {
		if c.Flavor != dash.FlavorNativeWindows {
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

	consumer.Infof("→ Launching inno uninstaller")

	// N.B: InnoSetup uninstallers are smart enough to elevate themselves.
	exitCode, err := installer.RunCommand(consumer, cmdTokens)
	err = installer.CheckExitCode(exitCode, err)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
