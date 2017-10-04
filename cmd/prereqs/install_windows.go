// +build windows

package prereqs

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/cmd/msi"
	"github.com/itchio/butler/comm"
	"github.com/natefinch/npipe"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
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

	hasConn := true
	// TODO: remove once itch v23 is dead
	conn, err := npipe.Dial(pipePath)
	if err != nil {
		comm.Warnf("Could not dial pipe %s", conn)
		hasConn = false
	}

	writeLine := func(contents []byte) error {
		if !hasConn {
			return nil
		}

		contents = append(contents, '\n')

		_, err = conn.Write(contents)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	doWriteState := func(taskName string, status string) error {
		msg := PrereqState{
			Type:   "state",
			Name:   taskName,
			Status: status,
		}
		comm.Result(&msg)

		contents, err := json.Marshal(&msg)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return writeLine(contents)
	}

	writeState := func(taskName string, status string) {
		err := doWriteState(taskName, status)
		if err != nil {
			switch err := err.(type) {
			case *errors.Error:
				comm.Warnf("Couldn't write log entry: %s", err.ErrorStack())
			default:
				comm.Warnf("Couldn't write log entry: %s", err.Error())
			}
		}
	}

	doLogf := func(format string, args ...interface{}) error {
		comm.Logf(format, args...)
		message := fmt.Sprintf(format, args...)

		contents, err := json.Marshal(&PrereqLogEntry{
			Type:    "log",
			Message: message,
		})
		if err != nil {
			return errors.Wrap(err, 0)
		}

		err = writeLine([]byte(contents))
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	logf := func(format string, args ...interface{}) {
		err := doLogf(format, args...)
		if err != nil {
			switch err := err.(type) {
			case *errors.Error:
				comm.Warnf("Couldn't write log entry: %s", err.ErrorStack())
			default:
				comm.Warnf("Couldn't write log entry: %s", err.Error())
			}
		}
	}

	logf("Installing %d prereqs", len(plan.Tasks))
	startTime := time.Now()

	var failed []string

	for _, task := range plan.Tasks {
		taskStartTime := time.Now()
		writeState(task.Name, "installing")

		logf("")
		logf("# Installing %s", task.Name)
		logf("")

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

			// TODO: deduplicate with doMsiInstall simply by redirecting log messages
			// to the named pipe
			err = msi.Install(ctx, commandPath, logPath, "")
			if err != nil {
				logf("MSI install failed: %s", err.Error())
				lf, openErr := os.Open(logPath)
				if openErr != nil {
					logf("And what's more, we can't open the log: %s", openErr.Error())
				} else {
					// grok UTF-16
					win16be := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
					// ...but abide by the BOM if there's one
					utf16bom := unicode.BOMOverride(win16be.NewDecoder())

					unicodeReader := transform.NewReader(lf, utf16bom)

					defer lf.Close()
					logf("Full MSI log follows:")
					s := bufio.NewScanner(unicodeReader)
					for s.Scan() {
						logf("[msi] %s", s.Text())
					}
					if scanErr := s.Err(); scanErr != nil {
						logf("While reading msi log: %s", scanErr.Error())
					}
				}

				failed = append(failed, task.Name)
			}
		} else {
			cmd := exec.Command(commandPath, args...)
			cmd.Dir = task.WorkDir

			logf("Launching %s %s", task.Info.Command, strings.Join(args, " "))

			err = cmd.Run()
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
						code := status.ExitStatus()
						known := false
						for _, exitCode := range task.Info.ExitCodes {
							if code == exitCode.Code {
								if exitCode.Success {
									logf("%s (Code %d), continuing", exitCode.Message, exitCode.Code)
								} else {
									logf("%s (Code %d), we'll error out eventually", exitCode.Message, exitCode.Code)
									failed = append(failed, task.Name)
								}
								known = true
							}
							break
						}

						if !known {
							logf("Got unknown exit code 0x%X (%d), will error out", code, code)
							failed = append(failed, task.Name)
						}
					} else {
						return errors.Wrap(err, 0)
					}
				} else {
					return errors.Wrap(err, 0)
				}
			}
		}

		writeState(task.Name, "done")
		logf("(Spent %s)", time.Since(taskStartTime))
	}

	logf("")
	if len(failed) > 0 {
		errMsg := fmt.Sprintf("Some prereqs failed to install: %s", strings.Join(failed, ", "))
		logf(errMsg)
		return errors.Wrap(errors.New(errMsg), 0)
	}

	logf("All done! (Spent %s total)", time.Since(startTime))

	return nil
}
