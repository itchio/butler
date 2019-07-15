package butlerd

import "path/filepath"

func GuessDBPath(appName string) string {
	if appName == "" {
		appName = "itch"
	}

	return filepath.Join(getAppPath(appName), "db", "butler.db")
}
