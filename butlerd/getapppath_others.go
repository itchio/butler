//+build !darwin

package butlerd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func getAppPath(appName string) string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		return filepath.Join(appData, appName)
	case "linux":
		configPath := os.Getenv("XDG_CONFIG_HOME")
		if configPath != "" {
			return filepath.Join(configPath, appName)
		} else {
			homePath := os.Getenv("HOME")
			return filepath.Join(homePath, ".config", appName)
		}
	}

	panic(fmt.Sprintf("unknown OS: %s", runtime.GOOS))
}
