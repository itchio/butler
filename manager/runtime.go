package manager

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/comm"
	itchio "github.com/itchio/go-itchio"
)

// Runtime describes an os-arch combo in a convenient way
type Runtime struct {
	Platform buse.ItchPlatform `json:"platform"`
	Is64     bool              `json:"is64"`
}

func (r *Runtime) String() string {
	var arch string
	if r.Is64 {
		arch = "64-bit"
	} else {
		arch = "32-bit"
	}
	return fmt.Sprintf("%s %s", arch, r.Platform)
}

func (r *Runtime) Arch() string {
	if r.Is64 {
		return "amd64"
	}
	return "386"
}

var cachedRuntime *Runtime

func CurrentRuntime() *Runtime {
	if cachedRuntime == nil {
		var is64 = is64Bit()
		var platform buse.ItchPlatform
		switch runtime.GOOS {
		case "linux":
			platform = buse.ItchPlatformLinux
		case "darwin":
			platform = buse.ItchPlatformOSX
		case "windows":
			platform = buse.ItchPlatformWindows
		default:
			platform = buse.ItchPlatformUnknown
		}

		cachedRuntime = &Runtime{
			Is64:     is64,
			Platform: platform,
		}
	}
	return cachedRuntime
}

var win64Arches = map[string]bool{
	"AMD64": true,
	"IA64":  true,
}

var hasDeterminedLinux64 = false
var cachedIsLinux64 bool

func is64Bit() bool {
	switch runtime.GOOS {
	case "darwin":
		// we don't ship for 32-bit mac
		return true
	case "linux":
		if !hasDeterminedLinux64 {
			cachedIsLinux64 = determineLinux64()
			hasDeterminedLinux64 = true
		}
		return cachedIsLinux64
	case "windows":
		// if we're currently running as a 64-bit executable then,
		// yeah, we're on 64-bit windows
		if runtime.GOARCH == "amd64" {
			return true
		}

		// otherwise, check environment variables
		// any value not in the map will return false (the zero value for bool ()
		return win64Arches[os.Getenv("PROCESSOR_ARCHITECTURE")] ||
			win64Arches[os.Getenv("PROCESSOR_ARCHITEW6432")]
	}

	// unsupported platform eh :(
	return false
}

func determineLinux64() bool {
	unameOutput, err := exec.Command("uname", "-m").Output()
	if err != nil {
		comm.Warnf("Could not determine if linux64 via uname: %s", err.Error())
	} else {
		return strings.TrimSpace(string(unameOutput)) == "x86_64"
	}

	archOutput, err := exec.Command("arch").Output()
	if err != nil {
		comm.Warnf("Could not determine if linux64 via uname: %s", err.Error())
	} else {
		return strings.TrimSpace(string(archOutput)) == "x86_64"
	}

	// if we're lacking uname AND arch, honestly, our chances are slim.
	// but in doubt, let's just assume the architecture of butler is the
	// same as the os
	comm.Warnf("Falling back to build architecture for linux64 detection")
	return runtime.GOARCH == "amd64"
}

type Platforms struct {
	Linux   bool
	Windows bool
	OSX     bool
	Android bool
}

func PlatformsForGame(game *itchio.Game) *Platforms {
	return &Platforms{
		Linux:   game.Linux,
		Windows: game.Windows,
		OSX:     game.OSX,
		Android: game.Android,
	}
}

func PlatformsForUpload(upload *itchio.Upload) *Platforms {
	return &Platforms{
		Linux:   upload.Linux,
		Windows: upload.Windows,
		OSX:     upload.OSX,
		Android: upload.Android,
	}
}

func (p *Platforms) IsCompatible(rt *Runtime) bool {
	switch rt.Platform {
	case buse.ItchPlatformLinux:
		return p.Linux
	case buse.ItchPlatformOSX:
		return p.OSX
	case buse.ItchPlatformWindows:
		return p.Windows
	}

	return false
}

// UploadExclusivityScore returns a higher value the closest an
// upload is to being *only for this platform*
func (p *Platforms) ExclusivityScore(rt *Runtime) int64 {
	var score int64 = 400

	switch rt.Platform {
	case buse.ItchPlatformLinux:
		if p.OSX {
			score -= 100
		}
		if p.Windows {
			score -= 100
		}
		if p.Android {
			score -= 200
		}
	case buse.ItchPlatformOSX:
		if p.Linux {
			score -= 100
		}
		if p.Windows {
			score -= 100
		}
		if p.Android {
			score -= 200
		}
	case buse.ItchPlatformWindows:
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
