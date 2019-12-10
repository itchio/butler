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

func BuildTime() *time.Time {
	if BuiltAt == "" {
		return nil
	}

	epoch, err := strconv.ParseInt(BuiltAt, 10, 64)
	if err != nil {
		return nil
	}

	t := time.Unix(epoch, 0)
	return &t
}

func buildVersionString() {
	timeString := "no build date"
	buildTime := BuildTime()
	if buildTime != nil {
		timeString = fmt.Sprintf("built on %s", buildTime.Format("Jan _2 2006 @ 15:04:05"))
	}

	VersionString = fmt.Sprintf("%s, %s", Version, timeString)

	if Commit != "" {
		VersionString = fmt.Sprintf("%s, ref %s", VersionString, Commit)
	}
}
