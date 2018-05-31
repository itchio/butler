package manager

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/ox"
)

type Platforms struct {
	Linux   bool
	Windows bool
	OSX     bool
	Android bool
}

func PlatformsForGame(game *itchio.Game) *Platforms {
	return &Platforms{
		Linux:   game.Traits.PlatformLinux,
		Windows: game.Traits.PlatformWindows,
		OSX:     game.Traits.PlatformOSX,
		Android: game.Traits.PlatformAndroid,
	}
}

func PlatformsForUpload(upload *itchio.Upload) *Platforms {
	return &Platforms{
		Linux:   upload.Traits.PlatformLinux,
		Windows: upload.Traits.PlatformWindows,
		OSX:     upload.Traits.PlatformOSX,
		Android: upload.Traits.PlatformAndroid,
	}
}

func (p *Platforms) IsCompatible(rt *ox.Runtime) bool {
	switch rt.Platform {
	case ox.PlatformLinux:
		return p.Linux
	case ox.PlatformOSX:
		return p.OSX
	case ox.PlatformWindows:
		return p.Windows
	}

	return false
}

// ExclusivityScore returns a higher value the closest an
// upload is to being *only for this platform*
func (p *Platforms) ExclusivityScore(rt *ox.Runtime) int64 {
	var score int64 = 400

	switch rt.Platform {
	case ox.PlatformLinux:
		if p.OSX {
			score -= 100
		}
		if p.Windows {
			score -= 100
		}
		if p.Android {
			score -= 200
		}
	case ox.PlatformOSX:
		if p.Linux {
			score -= 100
		}
		if p.Windows {
			score -= 100
		}
		if p.Android {
			score -= 200
		}
	case ox.PlatformWindows:
		if p.Linux {
			score -= 100
		}
		if p.OSX {
			score -= 100
		}
		if p.Android {
			score -= 200
		}
	default:
		score = 0
	}

	return score
}
