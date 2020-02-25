package manager

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
)

func IsCompatible(p itchio.Platforms, rt ox.Runtime) bool {
	switch rt.Platform {
	case ox.PlatformLinux:
		return p.Linux != ""
	case ox.PlatformOSX:
		return p.OSX != ""
	case ox.PlatformWindows:
		return p.Windows != ""
	}

	return false
}

// ExclusivityScore returns a higher value the closest an
// upload is to being *only for this platform*
func ExclusivityScore(p itchio.Platforms) int64 {
	rt := ox.CurrentRuntime()

	var score int64 = 400

	switch rt.Platform {
	case ox.PlatformLinux:
		if p.OSX != "" {
			score -= 100
		}
		if p.Windows != "" {
			score -= 100
		}
	case ox.PlatformOSX:
		if p.Linux != "" {
			score -= 100
		}
		if p.Windows != "" {
			score -= 100
		}
	case ox.PlatformWindows:
		if p.Linux != "" {
			score -= 100
		}
		if p.OSX != "" {
			score -= 100
		}
	default:
		score = 0
	}

	return score
}
