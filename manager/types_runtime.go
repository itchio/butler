package manager

import (
	"fmt"

	"github.com/itchio/ox"
)

// Runtimes

type SupportedRuntime struct {
	// os + arch, e.g. windows-i386, linux-amd64
	Runtime ox.Runtime `json:"runtime"`

	// wrapper tool (wine, etc.) that butler can launch itself
	Wrapper *Wrapper `json:"wrapper,omitempty"`

	RemoteLaunchName string `json:"remoteLaunchName,omitempty"`
}

func (sr SupportedRuntime) String() string {
	res := sr.Runtime.String()
	if sr.RemoteLaunchName != "" {
		res += fmt.Sprintf(" (remoteLaunchName=%s)", sr.RemoteLaunchName)
	} else if sr.Wrapper != nil {
		res += fmt.Sprintf(" (wrapper=%s)", sr.Wrapper.WrapperBinary)
	} else {
		res += " (native)"
	}
	return res
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
}
