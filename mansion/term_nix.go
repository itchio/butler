//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly
// +build linux darwin freebsd netbsd openbsd dragonfly

package mansion

import (
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

func IsTerminal() bool {
	return terminal.IsTerminal(syscall.Stdin)
}
