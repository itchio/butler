package launch

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/manager"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/headway/united"
	"github.com/itchio/hush/manifest"
	"github.com/itchio/ox"
	"github.com/itchio/pelican"
	"github.com/pkg/errors"
)

func ActionToLaunchTarget(consumer *state.Consumer, host manager.Host, installFolder string, manifestAction manifest.Action) (*butlerd.LaunchTarget, error) {
	if manifestAction.Platform == "" {
		manifestAction.Platform = host.Runtime.Platform
	}

	target := &butlerd.LaunchTarget{
		Action:   &manifestAction,
		Strategy: &butlerd.StrategyResult{},
		Host:     host,
	}

	// is it a path?
	fullPath := manifestAction.ExpandPath(manifestAction.Platform, installFolder)
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
		if host.Runtime.Platform == ox.PlatformOSX && strings.HasSuffix(strings.ToLower(fullPath), ".app") {
			consumer.Infof("(%s) is an app bundle, picking native strategy", fullPath)
			target.Strategy = &butlerd.StrategyResult{
				Strategy:       butlerd.LaunchStrategyNative,
				FullTargetPath: fullPath,
				Candidate:      appBundleCandidate(installFolder, fullPath, manifestAction.Path),
			}
			return target, nil
		}

		// if it's a folder, just browse it!
		consumer.Infof("(%s) is a folder, picking shell strategy", fullPath)
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
		consumer.Infof("(%s) yielded %d candidates when configured with dash", fullPath, len(verdict.Candidates))
		candidate := verdict.Candidates[0]
		target, err := CandidateToLaunchTarget(consumer, filepath.Dir(fullPath), host, candidate)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		target.Action = &manifestAction
		return target, nil
	}

	// must not be an executable, that's ok, just open it
	consumer.Infof("(%s) yielded no candidates, falling back to shell strategy", fullPath)
	target.Strategy = &butlerd.StrategyResult{
		Strategy:       butlerd.LaunchStrategyShell,
		FullTargetPath: fullPath,
	}
	return target, nil
}

// appBundleCandidate builds the candidate dash.Configure would yield for
// this bundle. Clients filter targets by candidate flavor, so manifest
// targets must carry one like heuristic targets do.
func appBundleCandidate(installFolder string, fullPath string, actionPath string) *dash.Candidate {
	relPath := actionPath
	if rel, err := filepath.Rel(installFolder, fullPath); err == nil {
		relPath = rel
	}
	relPath = filepath.ToSlash(relPath)
	return &dash.Candidate{
		Path:   relPath,
		Depth:  len(strings.Split(relPath, "/")),
		Flavor: dash.FlavorAppMacos,
	}
}

func CandidateToLaunchTarget(consumer *state.Consumer, basePath string, host manager.Host, candidate *dash.Candidate) (*butlerd.LaunchTarget, error) {
	fullPath := filepath.Join(basePath, filepath.FromSlash(candidate.Path))

	name := filepath.Base(fullPath)
	if candidate.Size > 0 {
		name += fmt.Sprintf(" (%s)", united.FormatBytes(candidate.Size))
	}

	target := &butlerd.LaunchTarget{
		Host: host,
		Action: &manifest.Action{
			Name: name,
			Path: candidate.Path,
		},
		Strategy: &butlerd.StrategyResult{
			Strategy:       flavorToStrategy(candidate.Flavor),
			FullTargetPath: fullPath,
			Candidate:      candidate,
		},
	}

	fallBackToShell := false

	if IsElevatedWindowsInstaller(consumer, candidate, fullPath) {
		consumer.Infof("(%s) is windows installer, falling back to shell strategy.", candidate.Path)
		fallBackToShell = true
	} else if target.Strategy.Strategy == butlerd.LaunchStrategyUnknown {
		consumer.Infof("(%s) unknown launch strategy, falling back to shell strategy.", candidate.Path)
		fallBackToShell = true
	}

	if fallBackToShell {
		target.Strategy.Strategy = butlerd.LaunchStrategyShell
		target.Strategy.FullTargetPath = filepath.Dir(fullPath)
	}
	return target, nil
}

// matchesRuntime mirrors dash's runtime matching: a payload is covered by
// its flavor, and a ROM also by "rom:<system>".
func matchesRuntime(candidate *dash.Candidate, runtimes []dash.Flavor) bool {
	if !candidate.IsPayload() {
		return false
	}
	system := ""
	if candidate.Engine != nil {
		system, _ = candidate.Engine.Details["system"].(string)
	}
	for _, r := range runtimes {
		if r == candidate.Flavor {
			return true
		}
		if candidate.Flavor == dash.FlavorROM && system != "" && r == dash.Flavor("rom:"+system) {
			return true
		}
	}
	return false
}

// runtimeTargetForAction hands a manifest action to the client when it
// runs what the action points at: a payload file, or a folder holding
// one. The action stays, with its name, arguments and settings; only the
// strategy changes. Anything else is returned as it was.
func runtimeTargetForAction(consumer *state.Consumer, host manager.Host, target *butlerd.LaunchTarget, runtimes []dash.Flavor) *butlerd.LaunchTarget {
	if len(runtimes) == 0 || target.Strategy == nil {
		return target
	}
	var basePath string
	var candidate *dash.Candidate
	switch target.Strategy.Strategy {
	case butlerd.LaunchStrategyNative:
		basePath = filepath.Dir(target.Strategy.FullTargetPath)
		candidate = target.Strategy.Candidate
	case butlerd.LaunchStrategyShell:
		basePath = target.Strategy.FullTargetPath
		verdict, err := dash.Configure(basePath, dash.ConfigureParams{
			Consumer: consumer,
			Filter:   filtering.FilterPaths,
		})
		if err != nil {
			consumer.Warnf("Could not configure (%s): %v", basePath, err)
			return target
		}
		for _, c := range verdict.Candidates {
			if matchesRuntime(c, runtimes) {
				candidate = c
				break
			}
		}
	default:
		return target
	}
	if candidate == nil || !matchesRuntime(candidate, runtimes) {
		return target
	}
	consumer.Infof("Action '%s' points at a %s payload the client runs itself", target.Action.Name, candidate.Flavor)
	runtimeTarget := RuntimeLaunchTarget(basePath, host, candidate)
	runtimeTarget.Action = target.Action
	return runtimeTarget
}

// RuntimeLaunchTarget describes a payload the client runs with its own
// runtime. Unlike the shell fallback, the target path is the payload
// itself, file or folder.
func RuntimeLaunchTarget(basePath string, host manager.Host, candidate *dash.Candidate) *butlerd.LaunchTarget {
	fullPath := filepath.Join(basePath, filepath.FromSlash(candidate.Path))

	name := filepath.Base(fullPath)
	if candidate.Size > 0 {
		name += fmt.Sprintf(" (%s)", united.FormatBytes(candidate.Size))
	}

	return &butlerd.LaunchTarget{
		Host: host,
		Action: &manifest.Action{
			Name: name,
			Path: candidate.Path,
		},
		Strategy: &butlerd.StrategyResult{
			Strategy:       butlerd.LaunchStrategyRuntime,
			FullTargetPath: fullPath,
			Candidate:      candidate,
		},
	}
}

func IsElevatedWindowsInstaller(consumer *state.Consumer, candidate *dash.Candidate, fullPath string) bool {
	if candidate.Flavor != dash.FlavorNativeWindows {
		return false
	}

	var memLines []string
	memConsumer := &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			memLines = append(memLines, fmt.Sprintf("[%s] %s", lvl, msg))
		},
	}

	f, err := os.Open(fullPath)
	if err != nil {
		consumer.Warnf("While opening candidate for pelican probe: %v", err)
		return false
	}

	peInfo, err := pelican.Probe(f, pelican.ProbeParams{
		Consumer: memConsumer,
	})
	if err != nil {
		consumer.Warnf("While opening candidate for pelican probe: %v", err)
		consumer.Warnf("Pelican log:\n%s", strings.Join(memLines, "\n"))
		return false
	}

	if peInfo.RequiresElevation() {
		consumer.Infof("(%s) Requires elevation", candidate.Path)
		return true
	}

	consumer.Infof("(%s) Does not require elevation", candidate.Path)
	return false
}

func flavorToPlatform(flavor dash.Flavor) *ox.Platform {
	switch flavor {
	// HTML
	case dash.FlavorHTML:
		return nil
	// Native
	case dash.FlavorNativeLinux:
		p := ox.PlatformLinux
		return &p
	case dash.FlavorNativeMacos:
		p := ox.PlatformOSX
		return &p
	case dash.FlavorNativeWindows:
		p := ox.PlatformWindows
		return &p
	case dash.FlavorAppMacos:
		p := ox.PlatformOSX
		return &p
	case dash.FlavorScript:
		return nil
	case dash.FlavorScriptWindows:
		p := ox.PlatformWindows
		return &p
	case dash.FlavorJar:
		return nil
	case dash.FlavorLove:
		return nil
	default:
		return nil
	}
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
