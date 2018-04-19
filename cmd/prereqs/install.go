package prereqs

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/cmd/elevate"
	"github.com/itchio/butler/installer"
	"github.com/itchio/ox"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

func (pc *PrereqsContext) InstallPrereqs(tsc *TaskStateConsumer, plan *PrereqPlan) error {
	consumer := pc.Consumer

	needElevation := false
	for _, task := range plan.Tasks {
		switch pc.Runtime.Platform {
		case ox.PlatformWindows:
			block := task.Info.Windows
			if block.Elevate {
				consumer.Infof("Will perform prereqs installation elevated because of (%s)", task.Name)
				needElevation = true
			}
		case ox.PlatformLinux:
			block := task.Info.Linux
			if len(block.EnsureSuidRoot) > 0 {
				consumer.Infof("Will perform prereqs installation elevated because (%s) has SUID binaries", task.Name)
				needElevation = true
			}
		}
	}

	planFile, err := ioutil.TempFile("", "butler-prereqs-plan.json")
	if err != nil {
		return errors.WithStack(err)
	}

	planPath := planFile.Name()
	defer os.Remove(planPath)

	enc := json.NewEncoder(planFile)
	err = enc.Encode(plan)
	if err != nil {
		return errors.WithStack(err)
	}

	err = planFile.Close()
	if err != nil {
		return errors.WithStack(err)
	}

	var args []string
	if needElevation {
		args = append(args, "--elevate")
	}
	args = append(args, []string{"install-prereqs", planPath}...)

	res, err := installer.RunSelf(&installer.RunSelfParams{
		Consumer: consumer,
		Args:     args,
		OnResult: func(value installer.Any) {
			switch value["type"] {
			case "state":
				{
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

					tsc.OnState(&butlerd.PrereqsTaskStateNotification{
						Name:   ps.Name,
						Status: ps.Status,
					})
				}
			}
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	if res.ExitCode != 0 {
		if res.ExitCode == elevate.ExitCodeAccessDenied {
			return errors.WithStack(butlerd.CodeOperationAborted)
		}
	}

	err = installer.CheckExitCode(res.ExitCode, err)
	if err != nil {
		return errors.WithStack(err)
	}

	// now to run some sanity checks (as regular user)
	for _, task := range plan.Tasks {
		switch pc.Runtime.Platform {
		case ox.PlatformLinux:
			block := task.Info.Linux
			for _, sc := range block.SanityChecks {
				err := pc.RunSanityCheck(task.Name, task.Info, sc)
				if err != nil {
					return errors.Wrapf(err, "sanity check failed for (%s)", task.Name)
				}
				consumer.Infof("Sanity check (%s ::: %s) passed", sc.Command, strings.Join(sc.Args, " ::: "))
			}
		}
	}

	return nil
}
