package nsis

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/elevate"
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
	cmd := []string{
		uninstallerPath,
		"/S", // run the uninstaller silently
	}

	pathArgs := getSeriouslyMisdesignedNsisPathArguments("_?=", params.InstallFolderPath)
	cmd = append(cmd, pathArgs...)

	consumer.Infof("launching nsis uninstaller, command:")
	consumer.Infof("%s", strings.Join(cmd, " ::: "))

	elevateParams := &elevate.ElevateParams{
		Command: cmd,
		Stdout:  makeConsumerWriter(consumer, "out"),
		Stderr:  makeConsumerWriter(consumer, "err"),
	}

	ret, err := elevate.Elevate(elevateParams)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if ret != 0 {
		return fmt.Errorf("non-zero exit code %d (%x)", ret, ret)
	}

	return nil
}
