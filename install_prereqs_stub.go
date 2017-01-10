// +build !windows

package main

import "fmt"

func installPrereqs(planPath string, pipePath string) {
	must(fmt.Errorf("install-prereqs is a windows-only command"))
}
