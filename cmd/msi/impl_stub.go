// +build !windows

package msi

import (
	"fmt"

	"github.com/itchio/butler/butler"
)

func Info(ctx *butler.Context, msiPath string) {
	return fmt.Errorf("msi-info is a windows-only command")
}

func ProductInfo(ctx *butler.Context, productCode string) error {
	return fmt.Errorf("msi-product-info is a windows-only command")
}

func Install(ctx *butler.Context, msiPath string, logPath string, target string) error {
	return fmt.Errorf("msi-install is a windows-only command")
}

func Uninstall(ctx *butler.Context, productCode string) error {
	return fmt.Errorf("msi-uninstall is a windows-only command")
}
