// +build !windows

package main

import "fmt"

func installPrereqs(planPath string, pipePath string) {
	must(fmt.Errorf("install-prereqs is a windows-only command"))
}

func testPrereqs(prereqs []string) {
	must(fmt.Errorf("test-prereqs is a windows-only command"))
}
