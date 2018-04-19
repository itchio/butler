package ox

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Platform string

// these coincide with the namings used in the itch.io backend
const (
	PlatformOSX     Platform = "osx"
	PlatformWindows Platform = "windows"
	PlatformLinux   Platform = "linux"
	PlatformUnknown Platform = "unknown"
)

// Runtime describes an os-arch combo in a convenient way
type Runtime struct {
	Platform Platform `json:"platform"`
	Is64     bool     `json:"is64"`
}

func (r *Runtime) String() string {
	var arch string
	if r.Is64 {
		arch = "64-bit"
	} else {
		arch = "32-bit"
	}
	var platform = "Unknown"
	switch r.Platform {
	case PlatformLinux:
		platform = "Linux"
	case PlatformOSX:
		platform = "macOS"
	case PlatformWindows:
		platform = "Windows"
	}
	return fmt.Sprintf("%s %s", arch, platform)
}

// OS returns the operating system in GOOS format
func (r *Runtime) OS() string {
	switch r.Platform {
	case PlatformLinux:
		return "linux"
	case PlatformOSX:
		return "darwin"
	case PlatformWindows:
		return "windows"
	default:
		return "unknown"
	}
}

// Arch returns the architecture in GOARCH format
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
		var platform Platform
		switch runtime.GOOS {
		case "linux":
			platform = PlatformLinux
		case "darwin":
			platform = PlatformOSX
		case "windows":
			platform = PlatformWindows
		default:
			platform = PlatformUnknown
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
	if err == nil {
		return strings.TrimSpace(string(unameOutput)) == "x86_64"
	}

	archOutput, err := exec.Command("arch").Output()
	if err == nil {
		return strings.TrimSpace(string(archOutput)) == "x86_64"
	}

	// if we're lacking uname AND arch, honestly, our chances are slim.
	// but in doubt, let's just assume the architecture of the current binary is the
	// same as the os
	return runtime.GOARCH == "amd64"
}
