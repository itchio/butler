// +build linux darwin freebsd netbsd openbsd dragonfly

package main

import (
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func isTerminal() bool {
	return terminal.IsTerminal(syscall.Stdin)
}
