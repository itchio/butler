package buildinfo

import (
	"fmt"
	"strconv"
	"time"
)

var (
	Version       = "head" // set by command-line on CI release builds
	BuiltAt       = ""     // set by command-line on CI release builds
	Commit        = ""     // set by command-line on CI release builds
	VersionString = ""     // formatted on boot from 'version' and 'builtAt'
)

func init() {
	buildVersionString()
}

func buildVersionString() {
	if BuiltAt != "" {
		epoch, err := strconv.ParseInt(BuiltAt, 10, 64)
		if err != nil {
			VersionString = fmt.Sprintf("%s, invalid build date", Version)
		} else {
			VersionString = fmt.Sprintf("%s, built on %s", Version, time.Unix(epoch, 0).Format("Jan _2 2006 @ 15:04:05"))
		}
	} else {
		VersionString = fmt.Sprintf("%s, no build date", Version)
	}
	if Commit != "" {
		VersionString = fmt.Sprintf("%s, ref %s", VersionString, Commit)
	}
}
