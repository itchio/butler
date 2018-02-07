package prereqs

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/installer"
	"github.com/itchio/wharf/state"
	"github.com/mitchellh/mapstructure"
)

func ElevatedInstall(consumer *state.Consumer, plan *PrereqPlan, tsc *TaskStateConsumer) error {
	planFile, err := ioutil.TempFile("", "butler-prereqs-plan.json")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	planPath := planFile.Name()
	defer os.Remove(planPath)

	enc := json.NewEncoder(planFile)
	err = enc.Encode(plan)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = planFile.Close()
	if err != nil {
		return errors.Wrap(err, 0)
	}

	res, err := installer.RunSelf(&installer.RunSelfParams{
		Consumer: consumer,
		Args: []string{
			"--elevate",
			"install-prereqs",
			planPath,
		},
		OnResult: func(value installer.Any) {
			if value["type"] == "state" {
				ps := &PrereqState{}
				msdec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
					TagName: "json",
					Result:  ps,
				})
				if err != nil {
					consumer.Warnf("could not decode result: %s", err.Error())
					return
				}

				err = msdec.Decode(value)
				if err != nil {
					consumer.Warnf("could not decode result: %s", err.Error())
					return
				}

				tsc.OnState(&buse.PrereqsTaskStateNotification{
					Name:   ps.Name,
					Status: ps.Status,
				})
			}
		},
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if res.ExitCode != 0 {
		if res.ExitCode == elevate.ExitCodeAccessDenied {
			return operate.ErrAborted
		}
	}

	err = installer.CheckExitCode(res.ExitCode, err)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
