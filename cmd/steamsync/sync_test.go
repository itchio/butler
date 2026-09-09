package steamsync

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "steam-sync.toml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestResolveEntries(t *testing.T) {
	dir := t.TempDir()
	path := writeConfig(t, dir, `
[[sync]]
app = 10
target = "leafo/a"
cache_dir = ".cache"

[[sync]]
app = 20
target = "leafo/b"
branch = "beta"
`)

	t.Run("config entries, resolved", func(t *testing.T) {
		entries, err := resolveEntries(syncRequest{Config: path})
		if err != nil || len(entries) != 2 {
			t.Fatalf("%v %v", entries, err)
		}
		if entries[0].CacheDir != filepath.Join(dir, ".cache") {
			t.Fatalf("cache dir not resolved: %q", entries[0].CacheDir)
		}
		if entries[1].Branch != "beta" {
			t.Fatalf("branch lost: %+v", entries[1])
		}
	})

	t.Run("config and app id together are refused", func(t *testing.T) {
		if _, err := resolveEntries(syncRequest{Config: path, AppID: 10, Target: "leafo/other"}); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("neither config nor app id", func(t *testing.T) {
		if _, err := resolveEntries(syncRequest{}); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("command line", func(t *testing.T) {
		entries, err := resolveEntries(syncRequest{AppID: 10, Target: "leafo/other", Branch: "public"})
		if err != nil || len(entries) != 1 || entries[0].Target != "leafo/other" || entries[0].CacheDir != "" {
			t.Fatalf("%v %v", entries, err)
		}
	})

	t.Run("per-app flags are refused in config mode", func(t *testing.T) {
		for name, req := range map[string]syncRequest{
			"hidden": {Config: path, Hidden: true},
			"branch": {Config: path, Branch: "beta"},
			"skip":   {Config: path, Skips: []uint32{1}},
			"map":    {Config: path, Mappings: []string{"1=win"}},
			"target": {Config: path, Target: "leafo/x"},
		} {
			if _, err := resolveEntries(req); err == nil {
				t.Errorf("%s: expected an error", name)
			}
		}
	})

	t.Run("cache-dir flag is the global cache dir in config mode", func(t *testing.T) {
		global := filepath.Join(t.TempDir(), "cache")
		entries, err := resolveEntries(syncRequest{Config: path, CacheDir: global, NoPush: true})
		if err != nil || len(entries) != 2 {
			t.Fatalf("%v %v", entries, err)
		}
		if entries[0].CacheDir != filepath.Join(dir, ".cache") {
			t.Fatalf("entry cache dir should win: %q", entries[0].CacheDir)
		}
		if entries[1].CacheDir != filepath.Join(global, "20") {
			t.Fatalf("entry should inherit the flag: %q", entries[1].CacheDir)
		}
	})

	t.Run("cache-dir flag overrides the file's global cache dir", func(t *testing.T) {
		withGlobal := writeConfig(t, t.TempDir(), "cache_dir = \"file-cache\"\n[[sync]]\napp = 1\ntarget = \"leafo/a\"\n")
		global := filepath.Join(t.TempDir(), "flag-cache")
		entries, err := resolveEntries(syncRequest{Config: withGlobal, CacheDir: global})
		if err != nil || len(entries) != 1 {
			t.Fatalf("%v %v", entries, err)
		}
		if entries[0].CacheDir != filepath.Join(global, "1") {
			t.Fatalf("flag should override the file: %q", entries[0].CacheDir)
		}
	})

	t.Run("command line needs a target", func(t *testing.T) {
		if _, err := resolveEntries(syncRequest{AppID: 10}); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("channel in target is refused everywhere", func(t *testing.T) {
		bad := writeConfig(t, t.TempDir(), "[[sync]]\napp = 1\ntarget = \"leafo/a:win\"\n")
		if _, err := resolveEntries(syncRequest{Config: bad}); err == nil {
			t.Fatal("config should refuse a channel target")
		}
		if _, err := resolveEntries(syncRequest{AppID: 1, Target: "leafo/a:win"}); err == nil {
			t.Fatal("command line should refuse a channel target")
		}
	})

	t.Run("no-push needs a cache dir per entry", func(t *testing.T) {
		if _, err := resolveEntries(syncRequest{Config: path, NoPush: true}); err == nil {
			t.Fatal("entry 20 has no cache dir")
		}
		if _, err := resolveEntries(syncRequest{AppID: 10, Target: "leafo/a", CacheDir: "x", NoPush: true}); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("missing config", func(t *testing.T) {
		if _, err := resolveEntries(syncRequest{Config: filepath.Join(dir, "nope.toml")}); err == nil {
			t.Fatal("expected an error")
		}
	})
}
