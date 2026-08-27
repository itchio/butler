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
