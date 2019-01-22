package msi

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/installer"
	"github.com/pkg/errors"
)

func (m *Manager) Uninstall(params installer.UninstallParams) error {
	consumer := params.Consumer
	receipt := params.Receipt

	if receipt == nil {
		return errors.New("Missing receipt, don't know what to uninstall")
	}

	if receipt.MSIProductCode == "" {
		return errors.New("Missing product code in receipt, don't know what to uninstall")
	}

	args := []string{
		"msi-uninstall",
		receipt.MSIProductCode,
	}

	consumer.Infof("Attempting non-elevated MSI uninstall")
	res, err := installer.RunSelf(&installer.RunSelfParams{
		Consumer: consumer,
		Args:     args,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if res.ExitCode != 0 {
		if shouldTryElevated(consumer, res) {
			args = append(args, "--elevate")

			consumer.Infof("Attempting elevated MSI uninstall")
			res, err := installer.RunSelf(&installer.RunSelfParams{
				Consumer: consumer,
				Args:     args,
			})
			if err != nil {
				return errors.WithStack(err)
			}

			if res.ExitCode != 0 {
				if res.ExitCode == elevate.ExitCodeAccessDenied {
					msg := "User or system did not grant elevation privileges"
					consumer.Errorf(msg)
					return errors.WithStack(butlerd.CodeOperationAborted)
				}

				consumer.Errorf("Elevated MSI uninstall failed (code %d, 0x%x), we're out of options", res.ExitCode, res.ExitCode)
				return errors.New("Elevated MSI uninstallation failed")
			}
		}
	}

	return nil
}
