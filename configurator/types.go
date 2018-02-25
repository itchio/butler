package configurator

// A Verdict contains a wealth of information on how to "launch" or "open" a specific
// folder.
type Verdict struct {
	// BasePath is the absolute path of the folder that was configured
	BasePath string `json:"basePath"`
	// TotalSize is the size in bytes of the folder and all its children, recursively
	TotalSize int64 `json:"totalSize"`
	// Candidates is a list of potentially interesting files, with a lot of additional info
	Candidates []*Candidate `json:"candidates"`
}

// A Candidate is a potentially interesting launch target, be it
// a native executable, a Java or Love2D bundle, an HTML index, etc.
type Candidate struct {
	// Path is relative to the configured folder
	Path string `json:"path"`
	// Mode describes file permissions
	Mode uint32 `json:"mode,omitempty"`
	// Depth is the number of path elements leading up to this candidate
	Depth int `json:"depth"`
	// Flavor is the type of a candidate - native, html, jar etc.
	Flavor Flavor `json:"flavor"`
	// Arch describes the architecture of a candidate (where relevant)
	Arch Arch `json:"arch,omitempty"`
	// Size is the size of the candidate's file, in bytes
	Size int64 `json:"size"`
	// Spell contains raw output from <https://github.com/fasterthanlime/wizardry>
	// @optional
	Spell []string `json:"spell,omitempty"`
	// WindowsInfo contains information specific to native Windows candidates
	// @optional
	WindowsInfo *WindowsInfo `json:"windowsInfo,omitempty"`
	// LinuxInfo contains information specific to native Linux candidates
	// @optional
	LinuxInfo *LinuxInfo `json:"linuxInfo,omitempty"`
	// MacosInfo contains information specific to native macOS candidates
	// @optional
	MacosInfo *MacosInfo `json:"macosInfo,omitempty"`
	// LoveInfo contains information specific to Love2D bundles (`.love` files)
	// @optional
	LoveInfo *LoveInfo `json:"loveInfo,omitempty"`
	// ScriptInfo contains information specific to shell scripts (`.sh`, `.bat` etc.)
	// @optional
	ScriptInfo *ScriptInfo `json:"scriptInfo,omitempty"`
	// JarInfo contains information specific to Java archives (`.jar` files)
	// @optional
	JarInfo *JarInfo `json:"jarInfo,omitempty"`
}

// Flavor describes whether we're dealing with a native executables, a Java archive, a love2d bundle, etc.
type Flavor string

const (
	// FlavorNativeLinux denotes native linux executables
	FlavorNativeLinux Flavor = "linux"
	// ExecNativeMacos denotes native macOS executables
	FlavorNativeMacos Flavor = "macos"
	// FlavorPe denotes native windows executables
	FlavorNativeWindows Flavor = "windows"
	// FlavorAppMacos denotes a macOS app bundle
	FlavorAppMacos Flavor = "app-macos"
	// FlavorScript denotes scripts starting with a shebang (#!)
	FlavorScript Flavor = "script"
	// FlavorScriptWindows denotes windows scripts (.bat or .cmd)
	FlavorScriptWindows Flavor = "windows-script"
	// FlavorJar denotes a .jar archive with a Main-Class
	FlavorJar Flavor = "jar"
	// FlavorHTML denotes an index html file
	FlavorHTML Flavor = "html"
	// FlavorLove denotes a love package
	FlavorLove Flavor = "love"
)

// The architecture of an executable
type Arch string

const (
	// 32-bit
	Arch386 Arch = "386"
	// 64-bit
	ArchAmd64 Arch = "amd64"
)

// Contains information specific to native windows executables
// or installer packages.
type WindowsInfo struct {
	// Particular type of installer (msi, inno, etc.)
	// @optional
	InstallerType WindowsInstallerType `json:"installerType,omitempty"`
	// True if we suspect this might be an uninstaller rather than an installer
	// @optional
	Uninstaller bool `json:"uninstaller,omitempty"`
	// Is this executable marked as GUI? This can be false and still pop a GUI, it's just a hint.
	// @optional
	Gui bool `json:"gui,omitempty"`
	// Is this a .NET assembly?
	// @optional
	DotNet bool `json:"dotNet,omitempty"`
}

// Which particular type of windows-specific installer
type WindowsInstallerType string

const (
	// Microsoft install packages (`.msi` files)
	WindowsInstallerTypeMsi WindowsInstallerType = "msi"
	// InnoSetup installers
	WindowsInstallerTypeInno WindowsInstallerType = "inno"
	// NSIS installers
	WindowsInstallerTypeNullsoft WindowsInstallerType = "nsis"
	// Self-extracting installers that 7-zip knows how to extract
	WindowsInstallerTypeArchive WindowsInstallerType = "archive"
)

// Contains information specific to native macOS executables
// or app bundles.
type MacosInfo struct {
}

// Contains information specific to native Linux executables
type LinuxInfo struct {
}

// Contains information specific to Love2D bundles
type LoveInfo struct {
	// The version of love2D required to open this bundle. May be empty
	// @optional
	Version string `json:"version,omitempty"`
}

// Contains information specific to shell scripts
type ScriptInfo struct {
	// Something like `/bin/bash`
	// @optional
	Interpreter string `json:"interpreter,omitempty"`
}

// Contains information specific to Java archives
type JarInfo struct {
	// The main Java class as specified by the manifest included in the .jar (if any)
	// @optional
	MainClass string `json:"mainClass,omitempty"`
}
