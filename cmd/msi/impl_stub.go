// +build !windows

package msi

import (
	"fmt"

	"github.com/itchio/butler/mansion"
)

func Info(ctx *mansion.Context, msiPath string) error {
	return fmt.Errorf("msi-info is a windows-only command")
}

func ProductInfo(ctx *mansion.Context, productCode string) error {
	return fmt.Errorf("msi-product-info is a windows-only command")
}

func Install(ctx *mansion.Context, msiPath string, logPath string, target string) error {
	return fmt.Errorf("msi-install is a windows-only command")
}

func Uninstall(ctx *mansion.Context, productCode string) error {
	return fmt.Errorf("msi-uninstall is a windows-only command")
}
