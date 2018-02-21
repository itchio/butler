// +build linux

package database

import (
	"os"
	"path/filepath"
)

// `%APPDATA%\itch` aka `C:\Users\foobar\AppData\Roaming\itch`
func GetAppDataPath(appName string) (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(os.Getenv("HOME"), ".config")
	}

	return filepath.Join(configHome, appName), nil
}
