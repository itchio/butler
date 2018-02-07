package installer

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/installer/loggerwriter"
	"github.com/itchio/wharf/state"
)

type Any map[string]interface{}

type RunSelfResult struct {
	ExitCode int
	Results  []Any
}

type OnResultFunc func(res Any)

type RunSelfParams struct {
	Consumer *state.Consumer
	Args     []string
	OnResult OnResultFunc
}

func RunSelf(params *RunSelfParams) (*RunSelfResult, error) {
	selfPath, err := os.Executable()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer := params.Consumer
	args := params.Args

	consumer.Infof("→ Invoking self:")
	consumer.Infof("  butler ::: %s", strings.Join(args, " ::: "))

	res := &RunSelfResult{
		Results: []Any{},
	}

	completeArgs := append([]string{"--json"}, args...)

	cmd := exec.Command(selfPath, completeArgs...)
	parser := newParserWriter(consumer, res, params.OnResult)
	cmd.Stdout = parser
	cmd.Stderr = loggerwriter.New(consumer, "err")

	err = cmd.Run()
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
				}
			} else {
				consumer.Infof("self: %s", string(data))
			}
		}
	}()

	return pw
}
