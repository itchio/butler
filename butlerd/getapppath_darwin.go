//+build darwin

package butlerd

import (
	"path/filepath"

	"github.com/itchio/ox/macox"
)

func getAppPath(appName string) string {
	appSupport, err := macox.GetApplicationSupportPath()
	if err != nil {
		panic(err)
	}
	return filepath.Join(appSupport, appName)
}
