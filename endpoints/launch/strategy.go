package launch

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/endpoints/launch/manifest"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/dash"
	"github.com/itchio/ox"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

type StrategyResult struct {
	Strategy       LaunchStrategy
	FullTargetPath string
	Candidate      *dash.Candidate
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

func DetermineStrategy(consumer *state.Consumer, runtime *ox.Runtime, installFolder string, manifestAction *butlerd.Action) (*StrategyResult, error) {
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
			return nil, errors.WithStack(err)
		}
		return nil, errors.WithStack(err)
	}

	if stats.IsDir() {
		// is it an app bundle?
		if runtime.Platform == ox.PlatformOSX && strings.HasSuffix(strings.ToLower(fullPath), ".app") {
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

	verdict, err := dash.Configure(fullPath, &dash.ConfigureParams{
		Consumer: consumer,
		Filter:   filtering.FilterPaths,
	})
	if err != nil {
		return nil, errors.WithStack(err)
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

func DetermineCandidateStrategy(basePath string, candidate *dash.Candidate) (*StrategyResult, error) {
	fullPath := filepath.Join(basePath, filepath.FromSlash(candidate.Path))

	res := &StrategyResult{
		Strategy:       flavorToStrategy(candidate.Flavor),
		FullTargetPath: fullPath,
		Candidate:      candidate,
	}
	return res, nil
}

func flavorToStrategy(flavor dash.Flavor) LaunchStrategy {
	switch flavor {
	// HTML
	case dash.FlavorHTML:
		return LaunchStrategyHTML
	// Native
	case dash.FlavorNativeLinux:
		return LaunchStrategyNative
	case dash.FlavorNativeMacos:
		return LaunchStrategyNative
	case dash.FlavorNativeWindows:
		return LaunchStrategyNative
	case dash.FlavorAppMacos:
		return LaunchStrategyNative
	case dash.FlavorScript:
		return LaunchStrategyNative
	case dash.FlavorScriptWindows:
		return LaunchStrategyNative
	case dash.FlavorJar:
		return LaunchStrategyNative
	case dash.FlavorLove:
		return LaunchStrategyNative
	default:
		return LaunchStrategyUnknown
	}
}
