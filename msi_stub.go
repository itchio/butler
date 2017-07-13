// +build !windows

package main

import "fmt"

func msiInfo(msiPath string) {
	must(fmt.Errorf("msi-info is a windows-only command"))
}

func msiProductInfo(productCode string) {
	must(fmt.Errorf("msi-product-info is a windows-only command"))
}

func msiInstall(msiPath string, logPath string, target string) {
	must(fmt.Errorf("msi-install is a windows-only command"))
}

func msiUninstall(productCode string) {
	must(fmt.Errorf("msi-uninstall is a windows-only command"))
}
