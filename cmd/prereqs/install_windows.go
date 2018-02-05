// +build windows

package prereqs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/installer"
	"github.com/itchio/butler/mansion"
)

func Install(ctx *mansion.Context, planPath string, pipePath string) error {
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

	namedPipe, err := NewNamedPipe(pipePath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	consumer := namedPipe.Consumer()

	consumer.Infof("Installing %d prereqs", len(plan.Tasks))
	startTime := time.Now()

	var failed []string

	for _, task := range plan.Tasks {
		taskStartTime := time.Now()
		namedPipe.WriteState(task.Name, "installing")

		consumer.Infof("")
		consumer.Infof("# Installing %s", task.Name)
		consumer.Infof("")

		commandPath := filepath.Join(task.WorkDir, task.Info.Command)
		args := task.Info.Args

		// MSI packages get special treatment for reasons.
		if strings.HasSuffix(strings.ToLower(task.Info.Command), ".msi") {
			tempDir, err := ioutil.TempDir("", "butler-msi-logs")
			if err != nil {
				return errors.Wrap(err, 0)
			}

			defer func() {
				os.RemoveAll(tempDir)
			}()

			logPath := filepath.Join(tempDir, "msi-install-log.txt")

			err = msi.Install(consumer, commandPath, logPath, "", nil)
			if err != nil {
				consumer.Errorf("MSI install failed: %s", err.Error())
				failed = append(failed, task.Name)
			}
		} else {
			cmdTokens := append([]string{commandPath}, args...)
			signedCode, err := installer.RunCommand(consumer, cmdTokens)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if signedCode != 0 {
				code := uint32(signedCode)
				known := false
				for _, exitCode := range task.Info.ExitCodes {
					if code == exitCode.Code {
						if exitCode.Success {
							consumer.Infof("%s (Code %d), continuing", exitCode.Message, exitCode.Code)
						} else {
							consumer.Warnf("%s (Code %d), we'll error out eventually", exitCode.Message, exitCode.Code)
							failed = append(failed, task.Name)
						}
						known = true
					}
					break
				}

				if !known {
					consumer.Infof("Got unknown exit code 0x%X (%d), will error out", code, code)
					failed = append(failed, task.Name)
				}
			}
		}

		namedPipe.WriteState(task.Name, "done")
		consumer.Infof("(Spent %s)", time.Since(taskStartTime))
	}

	consumer.Infof("")
	if len(failed) > 0 {
		errMsg := fmt.Sprintf("Some prereqs failed to install: %s", strings.Join(failed, ", "))
		consumer.Errorf(errMsg)
		return errors.Wrap(errors.New(errMsg), 0)
	}

	consumer.Statf("All done! (Spent %s total)", time.Since(startTime))

	return nil
}
