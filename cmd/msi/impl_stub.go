// +build !windows

package msi

import (
	"fmt"

	"github.com/itchio/wharf/state"
)

func Info(consumer *state.Consumer, msiPath string) (*MSIInfoResult, error) {
	return nil, fmt.Errorf("msi-info is a windows-only command")
}

func ProductInfo(consumer *state.Consumer, productCode string) (*MSIInfoResult, error) {
	return nil, fmt.Errorf("msi-product-info is a windows-only command")
}

func Install(consumer *state.Consumer, msiPath string, logPathIn string, target string, onError MSIErrorCallback) error {
	return fmt.Errorf("msi-install is a windows-only command")
}

func Uninstall(consumer *state.Consumer, productCode string, onError MSIErrorCallback) error {
	return fmt.Errorf("msi-uninstall is a windows-only command")
}
