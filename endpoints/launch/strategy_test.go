package launch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/hush/manifest"
	"github.com/itchio/ox"
)

func TestActionToLaunchTarget_AppBundleCarriesCandidate(t *testing.T) {
	t.Parallel()

	installFolder := t.TempDir()
	host := manager.Host{
		Runtime: ox.Runtime{Platform: ox.PlatformOSX, Is64: true},
	}
	consumer := &state.Consumer{}

	tests := []struct {
		name      string
		bundleRel string
		wantDepth int
	}{
		{"top-level bundle", "Game.app", 1},
		{"nested bundle", filepath.Join("Sub Dir", "Game.app"), 2},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			bundlePath := filepath.Join(installFolder, tt.bundleRel)
			if err := os.MkdirAll(bundlePath, 0o755); err != nil {
				t.Fatalf("creating app bundle dir: %v", err)
			}

			action := manifest.Action{
				Name: "Play",
				Path: tt.bundleRel,
			}
			target, err := ActionToLaunchTarget(consumer, host, installFolder, action)
			if err != nil {
				t.Fatalf("resolving app bundle action: %v", err)
			}

			if target.Strategy.Strategy != butlerd.LaunchStrategyNative {
				t.Errorf("strategy = %q, want %q", target.Strategy.Strategy, butlerd.LaunchStrategyNative)
			}
			if target.Strategy.FullTargetPath != bundlePath {
				t.Errorf("fullTargetPath = %q, want %q", target.Strategy.FullTargetPath, bundlePath)
			}

			c := target.Strategy.Candidate
			if c == nil {
				t.Fatalf("expected app bundle target to carry a candidate")
			}
			if c.Flavor != dash.FlavorAppMacos {
				t.Errorf("candidate flavor = %q, want %q", c.Flavor, dash.FlavorAppMacos)
			}
			wantPath := filepath.ToSlash(tt.bundleRel)
			if c.Path != wantPath {
				t.Errorf("candidate path = %q, want %q", c.Path, wantPath)
			}
			if c.Depth != tt.wantDepth {
				t.Errorf("candidate depth = %d, want %d", c.Depth, tt.wantDepth)
			}
		})
	}
}

func TestRuntimeTargetForAction(t *testing.T) {
	t.Parallel()

	installFolder := t.TempDir()
	loveDir := filepath.Join(installFolder, "game")
	if err := os.MkdirAll(loveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// dash knows an unpacked LÖVE game by its conf.lua
	for _, name := range []string{"main.lua", "conf.lua"} {
		if err := os.WriteFile(filepath.Join(loveDir, name), []byte("-- lua"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(installFolder, "game.pce"), make([]byte, 8192), 0o644); err != nil {
		t.Fatal(err)
	}

	host := manager.Host{Runtime: ox.Runtime{Platform: ox.PlatformLinux, Is64: true}}
	consumer := &state.Consumer{}
	resolve := func(path string, runtimes ...dash.Flavor) *butlerd.LaunchTarget {
		target, err := ActionToLaunchTarget(consumer, host, installFolder, manifest.Action{Name: "play", Path: path, Args: []string{"-x"}})
		if err != nil {
			t.Fatal(err)
		}
		return runtimeTargetForAction(consumer, host, target, runtimes)
	}

	// a folder with a LÖVE game inside, browsed until the client runs LÖVE
	if got := resolve("game").Strategy.Strategy; got != butlerd.LaunchStrategyShell {
		t.Errorf("folder without runtimes: strategy = %q, want shell", got)
	}
	target := resolve("game", "love")
	if target.Strategy.Strategy != butlerd.LaunchStrategyRuntime {
		t.Fatalf("folder with love runtime: strategy = %q, want runtime", target.Strategy.Strategy)
	}
	if target.Strategy.FullTargetPath != loveDir {
		t.Errorf("fullTargetPath = %q, want %q", target.Strategy.FullTargetPath, loveDir)
	}
	if target.Action == nil || target.Action.Name != "play" || len(target.Action.Args) != 1 {
		t.Errorf("action not kept: %+v", target.Action)
	}

	// a ROM file, native by dash's default strategy until the client runs it
	if got := resolve("game.pce", "rom:gba").Strategy.Strategy; got == butlerd.LaunchStrategyRuntime {
		t.Errorf("rom with another system's runtime became runtime")
	}
	target = resolve("game.pce", "rom:pce")
	if target.Strategy.Strategy != butlerd.LaunchStrategyRuntime {
		t.Fatalf("rom with its runtime: strategy = %q, want runtime", target.Strategy.Strategy)
	}
	if target.Strategy.FullTargetPath != filepath.Join(installFolder, "game.pce") {
		t.Errorf("fullTargetPath = %q", target.Strategy.FullTargetPath)
	}
}
