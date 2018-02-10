// +build !darwin

package macutil

import "errors"

func GetExecutablePath(bundlePath string) (string, error) {
	return "", errors.New("GetExecutablePath: only supported on macOS")
}
