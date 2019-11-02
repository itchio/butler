package installer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/installer/loggerwriter"
	"github.com/itchio/butler/manager"
	"github.com/itchio/headway/state"
	"github.com/itchio/httpkit/eos"
	"github.com/itchio/httpkit/eos/option"
	"github.com/itchio/ox"
	"github.com/itchio/savior"
	"github.com/itchio/savior/zipextractor"
	"github.com/pkg/errors"
)

type Any map[string]interface{}

type RunSelfResult struct {
	ExitCode int
	Results  []Any
	Errors   []string
}

type OnResultFunc func(res Any)

type RunSelfParams struct {
	Consumer *state.Consumer

	// Host to use to run self, may be non-native
	Host manager.Host

	// Required when we need to grab a non-native butler to run
	PrereqsDir string

	Args     []string
	OnResult OnResultFunc
}

type launchInfo struct {
	WorkingDir string
	SelfPath   string
}

func RunSelf(params RunSelfParams) (*RunSelfResult, error) {
	err := validation.ValidateStruct(&params,
		validation.Field(&params.PrereqsDir, validation.Required),
	)
	if err != nil {
		return nil, err
	}

	err = params.Host.Validate()
	if err != nil {
		return nil, err
	}

	var info launchInfo
	{
		selfPath, err := os.Executable()
		if err != nil {
			return nil, errors.Wrap(err, "getting path to own binary")
		}

		info = launchInfo{
			SelfPath:   selfPath,
			WorkingDir: "",
		}
	}

	if !params.Host.Runtime.Equals(ox.CurrentRuntime()) {
		switch params.Host.Runtime.Platform {
		case ox.PlatformWindows:
			err := fetchWindowsButler(params, &info)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Errorf("Don't know how to run self on non-native host %s", params.Host)
		}
	}

	consumer := params.Consumer
	args := params.Args

	consumer.Infof("→ Invoking self (%s)", info.SelfPath)
	consumer.Infof("  on host %s", params.Host)
	consumer.Infof("  butler ::: %s", strings.Join(args, " ::: "))

	args = append([]string{"--json"}, args...)

	if params.Host.Wrapper != nil {
		wr := params.Host.Wrapper

		// TODO: DRY (see launchers/native)
		var wrapperArgs []string
		wrapperArgs = append(wrapperArgs, wr.BeforeTarget...)
		if wr.NeedRelativeTarget {
			info.WorkingDir = filepath.Dir(info.SelfPath)
			relativeTarget := filepath.Base(info.SelfPath)
			wrapperArgs = append(wrapperArgs, relativeTarget)
		} else {
			wrapperArgs = append(wrapperArgs, info.SelfPath)
		}
		wrapperArgs = append(wrapperArgs, wr.BetweenTargetAndArgs...)
		wrapperArgs = append(wrapperArgs, args...)
		wrapperArgs = append(wrapperArgs, wr.AfterArgs...)
		args = wrapperArgs
		info.SelfPath = wr.WrapperBinary
	}

	res := &RunSelfResult{
		Results: []Any{},
	}

	cmd := exec.Command(info.SelfPath, args...)
	parser := newParserWriter(consumer, res, params.OnResult)
	cmd.Dir = info.WorkingDir
	cmd.Stdout = parser
	cmd.Stderr = loggerwriter.New(consumer, "err")

	consumer.Debugf("Final arguments: (%s)", strings.Join(args, " ::: "))
	consumer.Debugf("Final dir: (%s)", cmd.Dir)
	consumer.Debugf("Final info was: %#v", info)

	err = cmd.Run()
	if len(res.Errors) > 0 {
		return nil, errors.New(res.Errors[0])
	}

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				res.ExitCode = status.ExitStatus()
				return res, nil
			}
		}

		return nil, err
	}

	return res, nil
}

func fetchWindowsButler(params RunSelfParams, info *launchInfo) error {
	consumer := params.Consumer
	consumer.Debugf("Fetching butler for windows...")

	osarch := "windows-386"
	channel := osarch

	version := buildinfo.Version
	if buildinfo.Version == "head" {
		channel = osarch + "-head"
		version = buildinfo.Commit
	}

	if version == "" {
		// should only happen in development
		version = "LATEST"
	}
	consumer.Debugf("Channel: (%s)", channel)
	consumer.Debugf("Version: (%s)", version)

	installFolder := filepath.Join(params.PrereqsDir, osarch, "butler", version)
	consumer.Debugf("Will install to: (%s)", installFolder)
	url := fmt.Sprintf("https://broth.itch.ovh/butler/%s/%s/archive/default", channel, version)

	f, err := eos.Open(url, option.WithConsumer(consumer))
	if err != nil {
		return err
	}
	defer f.Close()

	stats, err := f.Stat()
	if err != nil {
		return err
	}

	sink := &savior.FolderSink{
		Directory: installFolder,
		Consumer:  consumer,
	}

	ze, err := zipextractor.New(f, stats.Size())
	if err != nil {
		return err
	}

	_, err = ze.Resume(nil, sink)
	if err != nil {
		return err
	}
	consumer.Debugf("Done installing")

	info.SelfPath = filepath.Join(installFolder, "butler.exe")

	return nil
}

func newParserWriter(consumer *state.Consumer, res *RunSelfResult, onResult OnResultFunc) io.Writer {
	pr, pw := io.Pipe()

	go func() {
		// note: we don't care terribly about bufio.Scanner error
		// conditions for this.
		s := bufio.NewScanner(pr)

		for s.Scan() {
			data := s.Bytes()

			obj := make(Any)
			err := json.Unmarshal(data, &obj)
			if err == nil {
				switch obj["type"] {
				case "log":
					if consumer.OnMessage != nil {
						if level, ok := obj["level"].(string); ok {
							if message, ok := obj["message"].(string); ok {
								consumer.OnMessage(level, message)
							}
						}
					}
				case "progress":
					if progress, ok := obj["progress"].(float64); ok {
						consumer.Progress(progress)
					}
				case "result":
					if value, ok := obj["value"].(map[string]interface{}); ok {
						if onResult != nil {
							onResult(value)
						} else {
							res.Results = append(res.Results, value)
						}
					} else {
						consumer.Warnf("runself: ignoring result because value is not a map")
					}
				case "error":
					if message, ok := obj["message"].(string); ok {
						res.Errors = append(res.Errors, message)
					}
				}
			} else {
				consumer.Infof("self: %s", string(data))
			}
		}
	}()

	return pw
}
