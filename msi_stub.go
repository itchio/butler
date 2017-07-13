// +build !windows

package main

import "fmt"

func msiInfo(msiPath string) {
	must(fmt.Errorf("msi-info is a windows-only command"))
}

func msiInstall(msiPath string) {
	must(fmt.Errorf("msi-install is a windows-only command"))
}
