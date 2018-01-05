package msi

import (
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/installer"
	"github.com/itchio/wharf/state"
	"github.com/mitchellh/mapstructure"
)

func shouldTryElevated(consumer *state.Consumer, res *installer.RunSelfResult) bool {
	for _, val := range res.Results {
		switch val["type"] {
		case "windowsInstallerError":
			var me msi.MSIWindowsInstallerError
			dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				Result: &me,
			})
			if err != nil {
				consumer.Errorf("Could not decode msi windows installer error: %s", err.Error())
				continue
			}

			dec.Decode(val["value"])
			if err != nil {
				consumer.Errorf("Could not decode msi windows installer error: %s", err.Error())
				continue
			}

			switch me.Code {
			case 1925:
				consumer.Infof("MSI complained about elevation, will retry elevated. Original error: ")
				consumer.Infof(me.Text)
				return true
			case 1721:
				consumer.Infof("MSI complained about a program it needs to run, will retry elevated. Original error: ")
				consumer.Infof(me.Text)
				return true
			}
		}
	}

	return false
}
