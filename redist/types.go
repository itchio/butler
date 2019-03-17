package redist

type RedistRegistry struct {
	Entries map[string]*RedistEntry `json:"entries"`
}

type RedistEntry struct {
	// FullName is the human-readable name for a redistributable
	FullName string `json:"fullName"`

	// Which platforms this redist is available for
	Platforms []string `json:"platforms"`

	// Version is the version of the redistributable we're distributing
	Version string `json:"version"`

	// Arch is the architecture of the redist in question
	Arch string `json:"arch"`

	// Windows-specific fields
	Windows *RedistEntryWindows `json:"windows,omitempty"`

	// Linux-specific fields
	Linux *RedistEntryLinux `json:"linux,omitempty"`

	// macOS-specific fields
	OSX *RedistEntryOSX `json:"osx,omitempty"`
}

type RedistEntryWindows struct {
	// Command is the exe/msi to fire up to install the redist
	Command string `json:"command"`

	// Elevate is true if the exe/msi needs administrative rights
	Elevate bool `json:"elevate"`

	// Args are passed to the Command
	Args []string `json:"args"`

	// RegistryKeys hint that the redist might already be installed, if present
	RegistryKeys []string `json:"registryKeys,omitempty"`

	// DLLs hint that the redist might already be installed, if we can load them
	DLLs []string `json:"dlls,omitempty"`

	// ExitCodes let prereqs installation succeed in case of non-zero exit codes
	// that mean something like "this is already installed"
	ExitCodes []*ExitCode `json:"exitCodes,omitempty"`
}

type RedistEntryLinux struct {
	// Is it something we download from the CDN and unpack ourselves or
	// do we use the system's package manager?
	Type LinuxRedistType `json:"type"`

	//---------------------------
	// Fields for 'hosted' type
	//---------------------------

	// List of files to `chmod +x`, relative to the prereqs destination folder
	EnsureExecutable []string `json:"ensureExecutable,omitempty"`

	// List of files to `chmod +x`, relative to the prereqs destination folder
	EnsureSuidRoot []string `json:"ensureSuidRoot,omitempty"`

	// Sanity checks to ensure that the redist is properly installed
	SanityChecks []*LinuxSanityCheck `json:"sanityChecks,omitempty"`
}

type LinuxSanityCheck struct {
	// Command to run (in prereqs destination folder)
	Command string `json:"command"`

	// Arguments to pass to the command
	Args []string `json:"args"`
}

type LinuxRedistType string

const (
	LinuxRedistTypeHosted = "hosted"
)

type RedistEntryOSX struct {
	// nothing so far
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
