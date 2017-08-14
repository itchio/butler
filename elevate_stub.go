// +build !windows

package main

import "fmt"

func elevate(command []string) {
	must(fmt.Errorf("elevate is a windows-only command"))
}

func pipe(command []string, stdin string, stdout string, stderr string) {
	must(fmt.Errorf("pipe is a windows-only command"))
}
