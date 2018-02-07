package msi

import (
	"fmt"

	"github.com/itchio/wharf/tlc"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/installer/bfs"
)

func (m *Manager) Install(params *installer.InstallParams) (*installer.InstallResult, error) {
	consumer := params.Consumer

	// we need the installer on disk to run it. this'll err if it's not,
	// and the caller is in charge of downloading it and calling us again.
	f, err := installer.AsLocalFile(params.File)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	angelParams := &bfs.SaveAngelsParams{
		Consumer: consumer,
		Folder:   params.InstallFolderPath,
		Receipt:  params.ReceiptIn,
	}

	cancel := make(chan struct{})
	defer close(cancel)
	bfs.StartAsymptoticProgress(consumer, cancel)

	infoRes, err := msi.Info(consumer, f.Name())
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	msiProductCode := infoRes.ProductCode

	angelResult, err := bfs.SaveAngels(angelParams, func() error {
		args := []string{
			"--elevate",
			"msi-install",
			f.Name(),
			"--target",
			params.InstallFolderPath,
		}

		consumer.Infof("Attempting elevated MSI install")
		res, err := installer.RunSelf(&installer.RunSelfParams{
			Consumer: consumer,
			Args:     args,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if res.ExitCode != 0 {
			if res.ExitCode == elevate.ExitCodeAccessDenied {
				msg := "User or system did not grant elevation privileges"
				consumer.Errorf(msg)
				return operate.ErrAborted
			}

			consumer.Errorf("Elevated MSI install failed (code %d, 0x%x), we're out of options", res.ExitCode, res.ExitCode)
			return errors.New("Elevated MSI installation failed, this package is probably not compatible")
		}

		consumer.Infof("MSI package installed successfully.")
		consumer.Infof("Making sure it installed in the directory we wanted...")
		container, err := tlc.WalkDir(params.InstallFolderPath, &tlc.WalkOpts{
			Filter: bfs.DotItchFilter(),
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if len(container.Files) == 0 {
			consumer.Errorf("No files were found in the install folder after install.")
			consumer.Errorf("The itch app won't be able to launch it.")

			var installLocation = "<unknown>"
			{
				infoRes2, err := msi.Info(consumer, f.Name())
				if err == nil && infoRes2.InstallLocation != "" {
					installLocation = infoRes2.InstallLocation
				}
			}
			consumer.Infof("Package install location: %s", installLocation)

			return fmt.Errorf("The MSI package was installed in an unexpected location: %s", installLocation)
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &installer.InstallResult{
		Files:          angelResult.Files,
		MSIProductCode: msiProductCode,
	}
	return res, nil
}
