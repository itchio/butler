package configurator

type Verdict struct {
	BasePath   string       `json:"basePath"`
	TotalSize  int64        `json:"totalSize"`
	Candidates []*Candidate `json:"candidates"`
}

// Candidate indicates what's interesting about a file
type Candidate struct {
	Path        string       `json:"path"`
	Mode        uint32       `json:"mode,omitempty"`
	Depth       int          `json:"depth"`
	Flavor      Flavor       `json:"flavor"`
	Arch        Arch         `json:"arch,omitempty"`
	Size        int64        `json:"size"`
	Spell       []string     `json:"spell,omitempty"`
	WindowsInfo *WindowsInfo `json:"windowsInfo,omitempty"`
	LinuxInfo   *LinuxInfo   `json:"linuxInfo,omitempty"`
	MacosInfo   *MacosInfo   `json:"macosInfo,omitempty"`
	LoveInfo    *LoveInfo    `json:"loveInfo,omitempty"`
	ScriptInfo  *ScriptInfo  `json:"scriptInfo,omitempty"`
	JarInfo     *JarInfo     `json:"jarInfo,omitempty"`
}

// Flavor describes the flavor of an executable
type Flavor string

const (
	// FlavorNativeLinux denotes native linux executables
	FlavorNativeLinux Flavor = "linux"
	// ExecNativeMacos denotes native macOS executables
	FlavorNativeMacos = "macos"
	// FlavorPe denotes native windows executables
	FlavorNativeWindows = "windows"
	// FlavorAppMacos denotes a macOS app bundle
	FlavorAppMacos = "app-macos"
	// FlavorScript denotes scripts starting with a shebang (#!)
	FlavorScript = "script"
	// FlavorScriptWindows denotes windows scripts (.bat or .cmd)
	FlavorScriptWindows = "windows-script"
	// FlavorJar denotes a .jar archive with a Main-Class
	FlavorJar = "jar"
	// FlavorHTML denotes an index html file
	FlavorHTML = "html"
	// FlavorLove denotes a love package
	FlavorLove = "love"
)

type Arch string

const (
	Arch386   Arch = "386"
	ArchAmd64      = "amd64"
)

type WindowsInfo struct {
	InstallerType WindowsInstallerType `json:"installerType,omitempty"`
	Uninstaller   bool                 `json:"uninstaller,omitempty"`
	Gui           bool                 `json:"gui,omitempty"`
	DotNet        bool                 `json:"dotNet,omitempty"`
}

type WindowsInstallerType string

const (
	WindowsInstallerTypeMsi      WindowsInstallerType = "msi"
	WindowsInstallerTypeInno                          = "inno"
	WindowsInstallerTypeNullsoft                      = "nsis"
	// self-extracting installer that unarchiver knows how to extract
	WindowsInstallerTypeArchive = "archive"
)

type MacosInfo struct {
}

type LinuxInfo struct {
}

type LoveInfo struct {
	Version string `json:"version,omitempty"`
}

type ScriptInfo struct {
	Interpreter string `json:"interpreter,omitempty"`
}

type JarInfo struct {
	MainClass string `json:"mainClass,omitempty"`
}
