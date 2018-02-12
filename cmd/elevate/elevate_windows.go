// +build windows

package elevate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/win32"
	"github.com/natefinch/npipe"
)

func Elevate(params *ElevateParams) (int, error) {
	command := params.Command

	if len(command) <= 0 {
		return -1, errors.New(`elevate needs a command to run`)
	}

	butlerExe, err := os.Executable()
	if err != nil {
		return -1, errors.Wrap(err, 0)
	}

	commandExe, err := findInPath(command[0])
	if err != nil {
		return -1, errors.Wrap(err, 0)
	}
	commandArgs := command[1:]

	pid := os.Getpid()

	stdoutPath := fmt.Sprintf(`\\.\pipe\elevate\%d\stdout`, pid)
	stdoutListener, err := npipe.Listen(stdoutPath)
	if err != nil {
		return -1, errors.Wrap(err, 0)
	}
	defer stdoutListener.Close()
	go relay(stdoutListener, params.Stdout)

	stderrPath := fmt.Sprintf(`\\.\pipe\elevate\%d\stderr`, pid)
	stderrListener, err := npipe.Listen(stderrPath)
	if err != nil {
		return -1, errors.Wrap(err, 0)
	}
	defer stderrListener.Close()
	go relay(stderrListener, params.Stderr)

	args := []string{"pipe", "--stdout", stdoutPath, "--stderr", stderrPath, "--"}
	args = append(args, commandExe)
	args = append(args, commandArgs...)

	wd, err := os.Getwd()
	if err != nil {
		return -1, errors.Wrap(err, 0)
	}

	err, code := win32.ShellExecuteAndWait(0, "runas", butlerExe, makeCmdLine(args), wd, syscall.SW_HIDE)
	if err != nil {
		if strings.Contains(err.Error(), "The operating system denied access to the specified file") {
			return ExitCodeAccessDenied, nil
		}
		return -1, errors.Wrap(err, 0)
	}

	return int(code), nil
}

func findInPath(commandExe string) (string, error) {
	// %PATH% may be different when running as Administrator,
	// so we need to resolve exe to an absolute path here
	if filepath.IsAbs(commandExe) {
		return commandExe, nil
	}

	binPaths := filepath.SplitList(os.Getenv("PATH"))

	for _, binPath := range binPaths {
		absPath := filepath.Join(binPath, commandExe)

		_, err := os.Stat(absPath)
		if err == nil {
			return absPath, nil
		}

		absPath = filepath.Join(binPath, commandExe+".exe")
		_, err = os.Stat(absPath)
		if err == nil {
			return absPath, nil
		}

		absPath = filepath.Join(binPath, commandExe+".cmd")
		_, err = os.Stat(absPath)
		if err == nil {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("Command '%s' not found in path", commandExe)
}

func makeCmdLine(args []string) string {
	var s string
	for _, v := range args {
		if s != "" {
			s += " "
		}
		s += syscall.EscapeArg(v)
	}
	return s
}

func relay(listener *npipe.PipeListener, output io.Writer) {
	conn, err := listener.Accept()
	if err != nil {
		return
	}

	io.Copy(output, conn)
}
