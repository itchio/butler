package launch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/manager"
	"github.com/itchio/dash"
	"github.com/itchio/headway/state"
	"github.com/itchio/hush/manifest"
	"github.com/itchio/ox"
)

func TestGetTargetsForHost_NativeAllManifestActionsFail(t *testing.T) {
	t.Parallel()

	installFolder := t.TempDir()
	runtime := ox.Runtime{Platform: ox.PlatformLinux, Is64: true}

	rc := &butlerd.RequestContext{Consumer: &state.Consumer{}}
	info := withInstallFolderInfo{
		installFolder: installFolder,
		runtime:       runtime,
	}
	host := manager.Host{Runtime: runtime}

	appManifest := &manifest.Manifest{
		Actions: manifest.Actions{
			{
				Name: "Default",
				Path: "Missing{{EXT}}",
			},
		},
	}

	_, err := getTargetsForHost(rc, nil, appManifest, &dash.Verdict{}, info, host, nil)
	if err == nil {
		t.Fatalf("expected error when all native manifest actions fail")
	}
	if !strings.Contains(err.Error(), "failed to resolve 1/1 manifest actions for native host") {
		t.Fatalf("expected native manifest failure error, got: %v", err)
	}
}

func TestGetTargetsForHost_NonNativeAllManifestActionsFail(t *testing.T) {
	t.Parallel()

	installFolder := t.TempDir()
	nativeRuntime := ox.Runtime{Platform: ox.PlatformLinux, Is64: true}
	wineRuntime := ox.Runtime{Platform: ox.PlatformWindows, Is64: false}

	rc := &butlerd.RequestContext{Consumer: &state.Consumer{}}
	info := withInstallFolderInfo{
		installFolder: installFolder,
		runtime:       nativeRuntime,
	}
	host := manager.Host{
		Runtime: wineRuntime,
		Wrapper: &manager.Wrapper{
			WrapperBinary: "wine",
		},
	}

	appManifest := &manifest.Manifest{
		Actions: manifest.Actions{
			{
				Name: "Default",
				Path: "Missing{{EXT}}",
			},
		},
	}

	targets, err := getTargetsForHost(rc, nil, appManifest, &dash.Verdict{}, info, host, nil)
	if err != nil {
		t.Fatalf("expected no error for non-native host when all manifest actions fail, got: %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("expected no targets from empty fallback verdict, got %d", len(targets))
	}
}

func TestGetTargetsForHost_NativePartialManifestResolution(t *testing.T) {
	t.Parallel()

	installFolder := t.TempDir()
	runtime := ox.Runtime{Platform: ox.PlatformLinux, Is64: true}

	rc := &butlerd.RequestContext{Consumer: &state.Consumer{}}
	info := withInstallFolderInfo{
		installFolder: installFolder,
		runtime:       runtime,
	}
	host := manager.Host{Runtime: runtime}

	validActionDir := filepath.Join(installFolder, "Sample Evil App")
	if err := os.MkdirAll(validActionDir, 0o755); err != nil {
		t.Fatalf("creating valid action dir: %v", err)
	}

	appManifest := &manifest.Manifest{
		Actions: manifest.Actions{
			{
				Name: "Valid",
				Path: "Sample Evil App{{EXT}}",
			},
			{
				Name: "Missing",
				Path: "Missing{{EXT}}",
			},
		},
	}

	targets, err := getTargetsForHost(rc, nil, appManifest, &dash.Verdict{}, info, host, nil)
	if err != nil {
		t.Fatalf("expected no error when at least one native action resolves, got: %v", err)
	}
	if len(targets) == 0 {
		t.Fatalf("expected at least one resolved target")
	}
}

func TestResolveSandbox(t *testing.T) {
	t.Parallel()

	yes := true
	no := false

	tests := []struct {
		name          string
		pref          *bool
		manifestOptIn bool
		want          bool
	}{
		{"no preference, no opt-in", nil, false, false},
		{"no preference, manifest opts in", nil, true, true},
		{"forced on", &yes, false, true},
		{"forced on, manifest also opts in", &yes, true, true},
		{"forced off", &no, false, false},
		{"forced off overrides manifest opt-in", &no, true, false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSandbox(tt.pref, tt.manifestOptIn)
			if got != tt.want {
				t.Errorf("resolveSandbox(%v, %v) = %v, want %v", tt.pref, tt.manifestOptIn, got, tt.want)
			}
		})
	}
}

func TestResolveSandboxOptions(t *testing.T) {
	t.Parallel()

	if got := resolveSandboxOptions(nil, butlerd.CaveSettings{}, nil); got != nil {
		t.Errorf("expected nil options when no tier says anything, got %+v", got)
	}

	yes := true
	no := false
	bubblewrap := butlerd.SandboxTypeBubblewrap

	noNetwork := true
	settings := butlerd.CaveSettings{
		SandboxNoNetwork: &noNetwork,
		SandboxAllowEnv:  &[]string{"DISPLAY"},
	}

	// an explicit params block replaces the lower tiers as a whole
	explicit := &butlerd.SandboxOptions{Type: butlerd.SandboxTypeFirejail}
	if got := resolveSandboxOptions(explicit, settings, nil); got != explicit {
		t.Errorf("expected explicit params to win as a whole, got %+v", got)
	}

	got := resolveSandboxOptions(nil, settings, nil)
	if got == nil || !got.NoNetwork || len(got.AllowEnv) != 1 || got.AllowEnv[0] != "DISPLAY" || got.Type != "" {
		t.Errorf("expected options built from settings, got %+v", got)
	}

	defaults := &butlerd.LaunchDefaults{
		SandboxType:      &bubblewrap,
		SandboxNoNetwork: &yes,
		SandboxAllowEnv:  []string{"PULSE_SERVER"},
	}

	got = resolveSandboxOptions(nil, butlerd.CaveSettings{}, defaults)
	if got == nil || got.Type != butlerd.SandboxTypeBubblewrap || !got.NoNetwork ||
		len(got.AllowEnv) != 1 || got.AllowEnv[0] != "PULSE_SERVER" {
		t.Errorf("expected options built from defaults, got %+v", got)
	}

	// a per-game override beats a default, including an explicit false
	settingsNoNetworkOff := butlerd.CaveSettings{SandboxNoNetwork: &no}
	got = resolveSandboxOptions(nil, settingsNoNetworkOff, defaults)
	if got == nil || got.NoNetwork {
		t.Errorf("expected settings' explicit false to beat the default true, got %+v", got)
	}

	// a present-but-empty per-game allowlist clears the default one
	settingsEmptyAllowEnv := butlerd.CaveSettings{SandboxAllowEnv: &[]string{}}
	got = resolveSandboxOptions(nil, settingsEmptyAllowEnv, defaults)
	if got == nil || len(got.AllowEnv) != 0 {
		t.Errorf("expected settings' empty allowlist to shadow the default, got %+v", got)
	}
}

func TestGetTargetsForHost_Runtimes(t *testing.T) {
	t.Parallel()

	installFolder := t.TempDir()
	rom := make([]byte, 16+16384)
	copy(rom, "NES\x1a")
	if err := os.WriteFile(filepath.Join(installFolder, "game.nes"), rom, 0644); err != nil {
		t.Fatal(err)
	}

	runtime := ox.Runtime{Platform: ox.PlatformLinux, Is64: true}
	rc := &butlerd.RequestContext{Consumer: &state.Consumer{}}
	info := withInstallFolderInfo{
		installFolder: installFolder,
		runtime:       runtime,
	}
	host := manager.Host{Runtime: runtime}

	verdict, err := dash.Configure(installFolder, dash.ConfigureParams{Consumer: &state.Consumer{}})
	if err != nil {
		t.Fatal(err)
	}

	// no runtimes: the lone payload falls back to the shell strategy, as before
	targets, err := getTargetsForHost(rc, nil, nil, verdict, info, host, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Strategy.Strategy != butlerd.LaunchStrategyShell {
		t.Fatalf("expected one shell target without runtimes, got %+v", targets)
	}

	// a runtime for another system changes nothing
	targets, err = getTargetsForHost(rc, nil, nil, verdict, info, host, []dash.Flavor{"rom:gba"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Strategy.Strategy != butlerd.LaunchStrategyShell {
		t.Fatalf("expected one shell target with a non-matching runtime, got %+v", targets)
	}

	for _, runtimes := range [][]dash.Flavor{{"rom:nes"}, {"rom"}} {
		targets, err = getTargetsForHost(rc, nil, nil, verdict, info, host, runtimes)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 {
			t.Fatalf("runtimes %v: expected one target, got %d", runtimes, len(targets))
		}
		s := targets[0].Strategy
		if s.Strategy != butlerd.LaunchStrategyRuntime {
			t.Fatalf("runtimes %v: expected runtime strategy, got %s", runtimes, s.Strategy)
		}
		if s.FullTargetPath != filepath.Join(installFolder, "game.nes") {
			t.Fatalf("runtimes %v: expected the ROM path, got %s", runtimes, s.FullTargetPath)
		}
		if s.Candidate == nil || s.Candidate.Flavor != dash.FlavorROM || s.Candidate.Engine.Details["system"] != "nes" {
			t.Fatalf("runtimes %v: candidate lacks ROM system: %+v", runtimes, s.Candidate)
		}
	}

	// a non-native host (wine, remote) does not get the client's runtimes
	other := manager.Host{Runtime: ox.Runtime{Platform: ox.PlatformWindows, Is64: true}}
	targets, err = getTargetsForHost(rc, nil, nil, verdict, info, other, []dash.Flavor{"rom"})
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Strategy.Strategy != butlerd.LaunchStrategyShell {
		t.Fatalf("expected shell target for non-native host, got %+v", targets)
	}
}
