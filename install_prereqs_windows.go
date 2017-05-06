package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"encoding/json"

	"os/exec"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/redist"
	"github.com/natefinch/npipe"
)

type PrereqTask struct {
	Name    string             `json:"name"`
	WorkDir string             `json:"workDir"`
	Info    redist.RedistEntry `json:"info"`
}

type PrereqPlan struct {
	Tasks []*PrereqTask `json:"tasks"`
}

type PrereqResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func installPrereqs(planPath string, pipePath string) {
	must(doInstallPrereqs(planPath, pipePath))
}

func doInstallPrereqs(planPath string, pipePath string) error {
	planReader, err := os.Open(planPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	decoder := json.NewDecoder(planReader)

	plan := &PrereqPlan{}
	err = decoder.Decode(plan)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Logf("Installing %d prereqs", len(plan.Tasks))
	startTime := time.Now()

	hasConn := true
	conn, err := npipe.Dial(pipePath)
	if err != nil {
		comm.Warnf("Could not dial pipe %s", conn)
		hasConn = false
	}

	writeState := func(taskName string, status string) error {
		if !hasConn {
			return nil
		}

		contents, err := json.Marshal(&PrereqResult{
			Name:   taskName,
			Status: status,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		contents = append(contents, '\n')

		_, err = conn.Write(contents)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	for _, task := range plan.Tasks {
		taskStartTime := time.Now()
		err = writeState(task.Name, "installing")
		if err != nil {
			comm.Warnf("Couldn't write installing state: %s", err.Error())
		}

		commandPath := filepath.Join(task.WorkDir, task.Info.Command)
		args := task.Info.Args

		// MSI packages get special treatment for reasons.
		if strings.HasSuffix(strings.ToLower(task.Info.Command), ".msi") {
			commandPath = "msiexec.exe"
			args = []string{
				"/quiet",
				"/norestart",
				"/i",
				filepath.Join(task.WorkDir, task.Info.Command),
			}
		}

		cmd := exec.Command(commandPath, args...)
		cmd.Dir = task.WorkDir

		err = cmd.Run()
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = writeState(task.Name, "done")
		if err != nil {
			comm.Warnf("Couldn't write done state: %s", err.Error())
		}

		comm.Logf("Done installing %s - took %s", task.Name, time.Since(taskStartTime))
	}

	comm.Statf("All done! Took %s", time.Since(startTime))

	return nil
}
