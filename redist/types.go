package redist

type RedistRegistry struct {
	Entries map[string]*RedistEntry `json:"entries"`
}

type RedistEntry struct {
	// FullName is the human-readable name for a redistributable
	FullName string `json:"fullName"`
	// Arch is the architecture of the redist in question
	Arch string `json:"arch"`
	// Command is the exe/msi to fire up to install the redist
	Command string `json:"command"`
	// Elevate is true if the exe/msi needs administrative rights
	Elevate bool `json:"elevate"`
	// Args are passed to the Command
	Args []string `json:"args"`
	// Version is the version of the redistributable we're distributing
	Version string `json:"version"`
	// RegistryKeys hint that the redist might already be installed, if present
	RegistryKeys []string `json:"registryKeys,omitempty"`
	// DLLs hint that the redist might already be installed, if we can load them
	DLLs []string `json:"dlls,omitempty"`
	// ExitCodes let prereqs installation succeed in case of non-zero exit codes
	// that mean something like "this is already installed"
	ExitCodes []*ExitCode `json:"exitCodes,omitempty"`

	Files []*File `json:"hashes"`
}

type File struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
}

type ExitCode struct {
	// Code is the process's exit code
	Code uint32 `json:"code"`
	// Success is true if that non-zero exit code means success
	Success bool `json:"success"`
	// Message is a human-readable message (in english) for what the exit code means
	Message string `json:"message"`
}
