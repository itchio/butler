package pelican

type Arch string

const (
	Arch386   = "386"
	ArchAmd64 = "amd64"
)

// PeInfo contains the architecture of a binary file
//
// For command `PeInfo`
type PeInfo struct {
	Arch                Arch                `json:"arch"`
	VersionProperties   map[string]string   `json:"versionProperties"`
	AssemblyInfo        *AssemblyInfo       `json:"assemblyInfo"`
	DependentAssemblies []*AssemblyIdentity `json:"dependentAssemblies"`
	Imports             []string            `json:"imports"`
}

func (pi *PeInfo) RequiresElevation() bool {
	if pi.AssemblyInfo == nil {
		return false
	}

	switch pi.AssemblyInfo.RequestedExecutionLevel {
	case "requireAdministrator", "highestAvailable":
		return true
	default:
		return false
	}
}

type AssemblyInfo struct {
	Identity    *AssemblyIdentity `json:"identity"`
	Description string            `json:"description"`

	RequestedExecutionLevel string `json:"requestedExecutionLevel,omitempty"`
}

type AssemblyIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"`

	ProcessorArchitecture string `json:"processorArchitecture,omitempty"`
	Language              string `json:"language,omitempty"`
	PublicKeyToken        string `json:"publicKeyToken,omitempty"`
}
