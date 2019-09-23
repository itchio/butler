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
	"github.com/itchio/headway/state"
	"github.com/itchio/headway/united"
	"github.com/itchio/ox"
	"github.com/pkg/errors"
)

func ActionToLaunchTarget(consumer *state.Consumer, platform ox.Platform, installFolder string, manifestAction *manifest.Action) (*butlerd.LaunchTarget, error) {
	actionCopy := *manifestAction
	manifestAction = &actionCopy
	if manifestAction.Platform == "" {
		manifestAction.Platform = platform
	}

	target := &butlerd.LaunchTarget{
		Action:   manifestAction,
		Strategy: &butlerd.StrategyResult{},
	}

	// is it a path?
	fullPath := manifest.ExpandPath(manifestAction, platform, installFolder)
	stats, err := os.Stat(fullPath)
	if err != nil {
		// is it a URL?
		{
			u, urlErr := url.Parse(manifestAction.Path)
			if urlErr == nil {
				if u.Scheme == "" {
					return nil, err
				}

				target.Strategy = &butlerd.StrategyResult{
					Strategy:       butlerd.LaunchStrategyURL,
					FullTargetPath: manifestAction.Path,
				}
				return target, nil
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
		if platform == ox.PlatformOSX && strings.HasSuffix(strings.ToLower(fullPath), ".app") {
			target.Strategy = &butlerd.StrategyResult{
				Strategy:       butlerd.LaunchStrategyNative,
				FullTargetPath: fullPath,
			}
			return target, nil
		}

		// if it's a folder, just browse it!
		target.Strategy = &butlerd.StrategyResult{
			Strategy:       butlerd.LaunchStrategyShell,
			FullTargetPath: fullPath,
		}
		return target, nil
	}

	verdict, err := dash.Configure(fullPath, dash.ConfigureParams{
		Consumer: consumer,
		Filter:   filtering.FilterPaths,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if len(verdict.Candidates) > 0 {
		candidate := verdict.Candidates[0]
		target, err := CandidateToLaunchTarget(filepath.Dir(fullPath), platform, candidate)
		if err != nil {
			target.Action = manifestAction
			return target, nil
		}
	}

	// must not be an executable, that's ok, just open it
	target.Strategy = &butlerd.StrategyResult{
		Strategy:       butlerd.LaunchStrategyShell,
		FullTargetPath: fullPath,
	}
	return target, nil
}

func CandidateToLaunchTarget(basePath string, platform ox.Platform, candidate *dash.Candidate) (*butlerd.LaunchTarget, error) {
	fullPath := filepath.Join(basePath, filepath.FromSlash(candidate.Path))

	name := filepath.Base(fullPath)
	if candidate.Size > 0 {
		name += fmt.Sprintf(" (%s)", united.FormatBytes(candidate.Size))
	}

	target := &butlerd.LaunchTarget{
		Action: &manifest.Action{
			Name: name,
			Path: candidate.Path,
		},
		Platform: platform,
		Strategy: &butlerd.StrategyResult{
			Strategy:       flavorToStrategy(candidate.Flavor),
			FullTargetPath: fullPath,
			Candidate:      candidate,
		},
	}
	return target, nil
}

func flavorToStrategy(flavor dash.Flavor) butlerd.LaunchStrategy {
	switch flavor {
	// HTML
	case dash.FlavorHTML:
		return butlerd.LaunchStrategyHTML
	// Native
	case dash.FlavorNativeLinux:
		return butlerd.LaunchStrategyNative
	case dash.FlavorNativeMacos:
		return butlerd.LaunchStrategyNative
	case dash.FlavorNativeWindows:
		return butlerd.LaunchStrategyNative
	case dash.FlavorAppMacos:
		return butlerd.LaunchStrategyNative
	case dash.FlavorScript:
		return butlerd.LaunchStrategyNative
	case dash.FlavorScriptWindows:
		return butlerd.LaunchStrategyNative
	case dash.FlavorJar:
		return butlerd.LaunchStrategyNative
	case dash.FlavorLove:
		return butlerd.LaunchStrategyNative
	default:
		return butlerd.LaunchStrategyUnknown
	}
}
