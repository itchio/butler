package main

import (
	"errors"
	"os"
	"strings"
	"syscall"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/win32"
)

func elevate(command []string) {
	if len(command) < 0 {
		must(errors.New(`elevate needs a command to run`))
	}
	comm.Logf("Should run: %s", strings.Join(command, " ::: "))

	exe := command[0]
	args := command[1:]

	wd, err := os.Getwd()
	must(err)

	err, code := win32.ShellExecuteAndWait(0, "runas", exe, makeCmdLine(args), wd, syscall.SW_NORMAL)
	must(err)

	os.Exit(int(code))
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

func pipe(command []string, stdin string, stdout string, stderr string) {
	comm.Logf("Should pipe: %s", strings.Join(command, " ::: "))
	comm.Logf("from stdin %s", stdin)
	comm.Logf("to stdout %s", stdout)
	comm.Logf("to stderr %s", stderr)
}
