// +build darwin

package database

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/runner/macutil"
)

// e.g. `~/Library/Application Support/itch`
func GetAppDataPath(appName string) (string, error) {
	appSupport, err := macutil.GetApplicationSupportPath()
	if err != nil {
		return "", errors.Wrap(err, 0)
	}

	return filepath.Join(appSupport, appName), nil
}
