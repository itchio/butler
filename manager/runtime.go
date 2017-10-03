package manager

import (
	"fmt"
	"runtime"

	itchio "github.com/itchio/go-itchio"
)

type ItchPlatform string

const (
	ItchPlatformOsx     ItchPlatform = "osx"
	ItchPlatformWindows              = "windows"
	ItchPlatformLinux                = "linux"
	ItchPlatformUnknown              = "unknown"
)

// Runtime describes an os-arch combo in a convenient way
type Runtime struct {
	Platform ItchPlatform `json:"platform"`
	Is64     bool         `json:"is64"`
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
		var is64 = runtime.GOARCH == "amd64"
		var platform ItchPlatform
		switch runtime.GOOS {
		case "linux":
			platform = ItchPlatformLinux
		case "darwin":
			platform = ItchPlatformOsx
		case "windows":
			platform = ItchPlatformWindows
		default:
			platform = ItchPlatformUnknown
		}

		cachedRuntime = &Runtime{
			Is64:     is64,
			Platform: platform,
		}
	}
	return cachedRuntime
}

func (rt *Runtime) UploadIsCompatible(game *itchio.Upload) bool {
	switch rt.Platform {
	case ItchPlatformLinux:
		return game.Linux
	case ItchPlatformOsx:
		return game.OSX
	case ItchPlatformWindows:
		return game.Windows
	}

	return false
}

func (rt *Runtime) GameIsCompatible(game *itchio.Game) bool {
	switch rt.Platform {
	case ItchPlatformLinux:
		return game.Linux
	case ItchPlatformOsx:
		return game.OSX
	case ItchPlatformWindows:
		return game.Windows
	}

	return false
}
