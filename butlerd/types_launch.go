package butlerd

import (
	"fmt"
	"strings"

	"github.com/itchio/butler/manager"
	"github.com/itchio/dash"
	"github.com/itchio/hush/manifest"
)

type LaunchTarget struct {
	// The manifest action corresponding to this launch target.
	// For implicit launch targets, a minimal one will be generated.
	Action *manifest.Action `json:"action"`

	// Host this launch target was found for
	Host manager.Host `json:"host"`

	// Detailed launch strategy
	Strategy *StrategyResult `json:"strategy"`
}

type StrategyResult struct {
	// Name of launch strategy used for launch target
	Strategy LaunchStrategy `json:"strategy"`

	// Absolute filesystem path of the target.
	FullTargetPath string `json:"fullTargetPath"`

	// If a local file, result of dash configure
	Candidate *dash.Candidate `json:"candidate"`
}

func (sr *StrategyResult) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("| (%s) (%s)", sr.FullTargetPath, sr.Strategy))
	if sr.Candidate != nil {
		lines = append(lines, sr.Candidate.String())
	}
	var explanation string
	switch sr.Strategy {
	case LaunchStrategyHTML:
		explanation = "☁ Will be opened as HTML5 app"
	case LaunchStrategyNative:
		explanation = "↗ Will be launched as a native application"
	case LaunchStrategyShell:
		explanation = "🗁 Will be opened in file manager"
	case LaunchStrategyURL:
		explanation = "🗏 Will be opened in browser, as web page"
	default:
		explanation = "(Unknown strategy)"
	}
	lines = append(lines, "|-- "+explanation)
	return strings.Join(lines, "\n")
}

type LaunchStrategy string

const (
	LaunchStrategyUnknown LaunchStrategy = ""
	LaunchStrategyNative  LaunchStrategy = "native"
	LaunchStrategyHTML    LaunchStrategy = "html"
	LaunchStrategyURL     LaunchStrategy = "url"
	LaunchStrategyShell   LaunchStrategy = "shell"
)
