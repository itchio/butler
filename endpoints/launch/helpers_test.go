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

	_, err := getTargetsForHost(rc, nil, appManifest, &dash.Verdict{}, info, host)
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

	targets, err := getTargetsForHost(rc, nil, appManifest, &dash.Verdict{}, info, host)
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

	targets, err := getTargetsForHost(rc, nil, appManifest, &dash.Verdict{}, info, host)
	if err != nil {
		t.Fatalf("expected no error when at least one native action resolves, got: %v", err)
	}
	if len(targets) == 0 {
		t.Fatalf("expected at least one resolved target")
	}
}
