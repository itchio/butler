package launch

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/configurator"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/manager"
	"github.com/itchio/wharf/state"
)

type StrategyResult struct {
	Strategy       LaunchStrategy
	FullTargetPath string
	Candidate      *configurator.Candidate
}

func (sr *StrategyResult) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("| (%s) (%s)", sr.FullTargetPath, sr.Strategy))
	if sr.Candidate != nil {
		lines = append(lines, sr.Candidate.String())
	}
	var explanation = ""
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

func DetermineStrategy(runtime *manager.Runtime, installFolder string, manifestAction *buse.Action) (*StrategyResult, error) {
	// is it a path?
	fullPath := manifest.ExpandPath(manifestAction, runtime, installFolder)
	stats, err := os.Stat(fullPath)
	if err != nil {
		// is it an URL?
		{
			u, urlErr := url.Parse(manifestAction.Path)
			if urlErr == nil {
				if u.Scheme == "" {
					return nil, err
				}

				res := &StrategyResult{
					Strategy:       LaunchStrategyURL,
					FullTargetPath: manifestAction.Path,
				}
				return res, nil
			}
		}

		if os.IsNotExist(err) {
			err = fmt.Errorf("Manifest action '%s' refers to non-existent path (%s)", manifestAction.Name, fullPath)
			return nil, errors.Wrap(err, 0)
		}
		return nil, errors.Wrap(err, 0)
	}

	if stats.IsDir() {
		// is it an app bundle?
		if runtime.Platform == buse.ItchPlatformOSX && strings.HasSuffix(strings.ToLower(fullPath), ".app") {
			res := &StrategyResult{
				Strategy:       LaunchStrategyNative,
				FullTargetPath: fullPath,
			}
			return res, nil
		}

		// if it's a folder, just browse it!
		res := &StrategyResult{
			Strategy:       LaunchStrategyShell,
			FullTargetPath: fullPath,
		}
		return res, nil
	}

	verdict, err := manager.Configure(&state.Consumer{}, fullPath, runtime)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if len(verdict.Candidates) > 0 {
		return DetermineCandidateStrategy(filepath.Dir(fullPath), verdict.Candidates[0])
	}

	// must not be an executable, that's ok, just open it
	res := &StrategyResult{
		Strategy:       LaunchStrategyShell,
		FullTargetPath: fullPath,
	}
	return res, nil
}

func DetermineCandidateStrategy(basePath string, candidate *configurator.Candidate) (*StrategyResult, error) {
	fullPath := filepath.Join(basePath, filepath.FromSlash(candidate.Path))

	res := &StrategyResult{
		Strategy:       flavorToStrategy(candidate.Flavor),
		FullTargetPath: fullPath,
		Candidate:      candidate,
	}
	return res, nil
}

func flavorToStrategy(flavor configurator.Flavor) LaunchStrategy {
	switch flavor {
	// HTML
	case configurator.FlavorHTML:
		return LaunchStrategyHTML
	// Native
	case configurator.FlavorNativeLinux:
		return LaunchStrategyNative
	case configurator.FlavorNativeMacos:
		return LaunchStrategyNative
	case configurator.FlavorNativeWindows:
		return LaunchStrategyNative
	case configurator.FlavorAppMacos:
		return LaunchStrategyNative
	case configurator.FlavorScript:
		return LaunchStrategyNative
	case configurator.FlavorScriptWindows:
		return LaunchStrategyNative
	case configurator.FlavorJar:
		return LaunchStrategyNative
	case configurator.FlavorLove:
		return LaunchStrategyNative
	default:
		return LaunchStrategyUnknown
	}
}
