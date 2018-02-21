// +build windows

package database

import (
	"os"
	"path/filepath"
)

// `%APPDATA%\itch` aka `C:\Users\foobar\AppData\Roaming\itch`
func GetAppDataPath(appName string) (string, error) {
	appData := os.Getenv("APPDATA")
	return filepath.Join(appData, appName), nil
}
