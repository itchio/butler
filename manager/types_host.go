package manager

import (
	"github.com/itchio/ox"
)

// Hosts

type Host struct {
	// os + arch, e.g. windows-i386, linux-amd64
	Runtime ox.Runtime `json:"runtime"`

	// wrapper tool (wine, etc.) that butler can launch itself
	Wrapper *Wrapper `json:"wrapper,omitempty"`

	RemoteLaunchName string `json:"remoteLaunchName,omitempty"`
}

type Wrapper struct {
	// wrapper {HERE} game.exe --launch-editor
	BeforeTarget []string `json:"beforeTarget"`
	// wrapper game.exe {HERE} --launch-editor
	BetweenTargetAndArgs []string `json:"betweenTargetAndArgs"`
	// wrapper game.exe --launch-editor {HERE}
	AfterArgs []string `json:"afterArgs"`

	// full path to the wrapper, like "wine"
	WrapperBinary string `json:"wrapperBinary"`

	// additional environment variables
	Env map[string]string `json:"env"`

	// When this is true, the wrapper can't function like this:
	//
	//   $ wine /path/to/game.exe
	//
	// It needs to function like this:
	//
	//   $ cd /path/to
	//   $ wine game.exe
	//
	// This is at least true for wine, which cannot find required DLLs
	// otherwise. This might be true for other wrappers, so it's an option here.
	NeedRelativeTarget bool `json:"needRelativeTarget"`
}
