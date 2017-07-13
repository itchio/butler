package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"encoding/json"

	"os/exec"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/redist"
	"github.com/natefinch/npipe"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// PrereqTask describes something the prereq installer has to do
type PrereqTask struct {
	Name    string             `json:"name"`
	WorkDir string             `json:"workDir"`
	Info    redist.RedistEntry `json:"info"`
}

// PrereqPlan contains a list of tasks for the prereq installer
type PrereqPlan struct {
	Tasks []*PrereqTask `json:"tasks"`
}

// PrereqState informs the caller on the current status of a prereq
type PrereqState struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PrereqLogEntry sends an information to the caller on the progress of the task
type PrereqLogEntry struct {
	Type    string `json:"type"`
	Message string `json:"message"`
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

	hasConn := true
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
		contents, err := json.Marshal(&PrereqState{
			Type:   "state",
			Name:   taskName,
			Status: status,
		})
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

			err = doMsiInstall(commandPath, logPath, "")
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
									logf("%s, continuing", exitCode.Message)
								} else {
									logf("%s, we'll error out eventually", exitCode.Message)
									failed = append(failed, task.Name)
								}
								known = true
							}
							break
						}

						if !known {
							logf("Got unknown exit code %d, will error out", code)
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

func testPrereqs(prereqs []string) {
	must(doTestPrereqs(prereqs))
}

func doTestPrereqs(prereqs []string) error {
	comm.Opf("Fetching registry...")

	baseURL := "https://dl.itch.ovh/itch-redists"

	infoURL := fmt.Sprintf("%s/info.json?t=%d", baseURL, time.Now().Unix())
	res, err := http.Get(infoURL)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if res.StatusCode != 200 {
		return errors.Wrap(fmt.Errorf("While getting redist registry, got HTTP %d", res.StatusCode), 0)
	}

	dec := json.NewDecoder(res.Body)

	registry := &redist.RedistRegistry{}
	err = dec.Decode(registry)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	if len(prereqs) == 0 {
		comm.Logf("")
		comm.Statf("No prereqs specified, here are those we know about: \n")

		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoFormatHeaders(false)
		table.SetColWidth(60)
		table.SetHeader([]string{"Name", "Arch", "Description", "Version"})
		for name, info := range registry.Entries {
			table.Append([]string{name, info.Arch, info.FullName, info.Version})
		}
		table.Render()
		return nil
	}

	if len(prereqs) == 1 && prereqs[0] == "all" {
		prereqs = nil
		for name := range registry.Entries {
			prereqs = append(prereqs, name)
		}
	}

	comm.Logf("Testing out prereqs %s", strings.Join(prereqs, ", "))

	plan := &PrereqPlan{}

	tempDir := filepath.Join(os.TempDir(), "butler-test-prereqs")
	comm.Logf("Working in %s", tempDir)
	comm.Logf("(This helps not having to re-download the prereqs between runs, but feel free to wipe it)")

	err = os.MkdirAll(tempDir, 0755)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	for _, name := range prereqs {
		info, ok := registry.Entries[name]
		if !ok {
			comm.Warnf("Unknown prereq %s, skipping", name)
			continue
		}

		comm.Opf("Downloading prereq %s", name)

		workDir := filepath.Join(tempDir, name)
		err = os.MkdirAll(workDir, 0755)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		task := &PrereqTask{
			Info:    *info,
			Name:    name,
			WorkDir: workDir,
		}

		url := fmt.Sprintf("%s/%s/%s", baseURL, name, info.Command)
		dest := filepath.Join(workDir, info.Command)
		_, err = tryDl(url, dest)
		if err != nil {
			comm.Logf("Could not donwload prereq %s", name)
			return errors.Wrap(err, 0)
		}

		plan.Tasks = append(plan.Tasks, task)
	}

	planPath := filepath.Join(tempDir, "butler_install_plan.json")
	comm.Logf("Writing plan to %s", planPath)

	planContents, err := json.Marshal(plan)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	err = ioutil.WriteFile(planPath, planContents, 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	comm.Opf("Handing off to install-prereqs...")

	err = doInstallPrereqs(planPath, "")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
