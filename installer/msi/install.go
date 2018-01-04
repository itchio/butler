package msi

import (
	"fmt"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/comm"
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

	var msiProductCode string

	angelResult, err := bfs.SaveAngels(angelParams, func() error {
		infoRes, err := msi.Info(consumer, f.Name())
		if err != nil {
			return errors.Wrap(err, 0)
		}

		msiProductCode = infoRes.ProductCode

		var msiErrors []msi.MSIWindowsInstallerError

		onError := func(me msi.MSIWindowsInstallerError) {
			msiErrors = append(msiErrors, me)
		}

		err = msi.Install(consumer, f.Name(), "", params.InstallFolderPath, onError)
		if err != nil {
			comm.Warnf("MSI installation failed: %s", err.Error())

			for _, me := range msiErrors {
				comm.Warnf(me.Text)
			}

			// try to make a nice error:
			if len(msiErrors) > 0 {
				var errorStrings []string
				for _, me := range msiErrors {
					errorStrings = append(errorStrings, me.Text)
				}
				return fmt.Errorf("MSI installation failed:\n%s", strings.Join(errorStrings, "\n"))
			}
			return fmt.Errorf("MSI installation failed: %s", err.Error())
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
