package launch

type LaunchStrategy string

const (
	LaunchStrategyUnknown LaunchStrategy = ""
	LaunchStrategyNative  LaunchStrategy = "native"
	LaunchStrategyHTML    LaunchStrategy = "html"
	LaunchStrategyURL     LaunchStrategy = "url"
	LaunchStrategyShell   LaunchStrategy = "shell"
)

type LauncherParams struct {
	WorkingDirectory string

	// If relative, it's relative to the WorkingDirectory
	Path string

	// If true, enable sandbox
	Sandbox bool

	// Additional command-line arguments
	Args []string

	// Additional environment variables
	Env map[string]string

	// $TEMP
	TempDir string
}
