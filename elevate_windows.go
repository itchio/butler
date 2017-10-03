package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/win32"
	"github.com/natefinch/npipe"
)

func relay(listener *npipe.PipeListener, output io.Writer) {
	conn, err := listener.Accept()
	if err != nil {
		return
	}

	io.Copy(output, conn)
}

func elevate(command []string) {
	if len(command) < 0 {
		comm.Dief(`elevate needs a command to run`)
	}

	butlerExe, err := os.Executable()
	must(err)

	commandExe, err := findInPath(command[0])
	must(err)
	commandArgs := command[1:]

	pid := os.Getpid()

	stdoutPath := fmt.Sprintf(`\\.\pipe\elevate\%d\stdout`, pid)
	stdoutListener, err := npipe.Listen(stdoutPath)
	must(err)
	defer stdoutListener.Close()
	go relay(stdoutListener, os.Stdout)

	stderrPath := fmt.Sprintf(`\\.\pipe\elevate\%d\stderr`, pid)
	stderrListener, err := npipe.Listen(stderrPath)
	must(err)
	defer stderrListener.Close()
	go relay(stderrListener, os.Stderr)

	args := []string{"pipe", "--stdout", stdoutPath, "--stderr", stderrPath, "--"}
	args = append(args, commandExe)
	args = append(args, commandArgs...)

	wd, err := os.Getwd()
	must(err)

	err, code := win32.ShellExecuteAndWait(0, "runas", butlerExe, makeCmdLine(args), wd, syscall.SW_HIDE)
	must(err)

	os.Exit(int(code))
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

// GetPipeFunc is a function type that gets an io.ReadCloser and can err
type GetPipeFunc func() (io.ReadCloser, error)

func pipe(command []string, stdin string, stdout string, stderr string) {
	cmd := exec.Command(command[0], command[1:]...)

	hook := func(namedPath string, fallback *os.File) io.Writer {
		pipe, err := npipe.DialTimeout(namedPath, 1*time.Second)
		if err != nil {
			return fallback
		}
		return pipe
	}

	cmd.Stdout = hook(stdout, os.Stdout)
	cmd.Stderr = hook(stderr, os.Stderr)

	exitCode := 0

	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if stat, ok := ee.ProcessState.Sys().(syscall.WaitStatus); ok {
				exitCode = int(stat.ExitCode)
			}
		} else {
			fmt.Fprintf(cmd.Stderr, "While running %s: %s", command[0], err.Error())
			exitCode = 1
			must(err)
		}
	}

	os.Exit(exitCode)
}
