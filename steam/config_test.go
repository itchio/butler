package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSyncConfigLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "steam-sync.toml")
	os.WriteFile(path, []byte(`
[[sync]]
app = 412870
target = "leafo/other"
branch = "beta"
skip = [412875]
cache_dir = ".cache"

[sync.map]
412871 = "win-64"

[[sync]]
app = 3838630
target = "leafo/x-moon"
`), 0o644)

	c, err := LoadSyncConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Sync) != 2 {
		t.Fatalf("want 2 entries, got %d", len(c.Sync))
	}
	e := c.Find(412870)
	if e == nil || e.Branch != "beta" || e.CacheDir != ".cache" {
		t.Fatalf("entry should keep its cache dir as written: %+v", e)
	}
	if c.Resolve(*e).CacheDir != filepath.Join(dir, ".cache") {
		t.Fatalf("resolve: %+v", c.Resolve(*e))
	}
	opts, err := e.PlanOptions("pw")
	if err != nil || opts.Map[412871] != "win-64" || !opts.Skip[412875] || opts.Password != "pw" {
		t.Fatalf("plan options: %+v %v", opts, err)
	}
	if c.Find(3838630).BranchOrDefault() != "public" {
		t.Fatal("branch should default to public")
	}

	if e.Map["412871"] != "win-64" {
		t.Fatal("map lost")
	}
}

func TestSyncConfigRejectsBadEntries(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"missing target":                 "[[sync]]\napp = 1\n",
		"duplicate app":                  "[[sync]]\napp = 1\ntarget = \"a/b\"\n[[sync]]\napp = 1\ntarget = \"a/c\"\n",
		"bad map key":                    "[[sync]]\napp = 1\ntarget = \"a/b\"\n[sync.map]\nabc = \"win\"\n",
		"entry cache_dir same as global": "cache_dir = \"cache\"\n[[sync]]\napp = 1\ntarget = \"a/b\"\ncache_dir = \"./cache/\"\n",
	} {
		path := filepath.Join(dir, name+".toml")
		os.WriteFile(path, []byte(body), 0o644)
		if _, err := LoadSyncConfig(path); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestSyncConfigGlobalCacheDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "steam-sync.toml")
	os.WriteFile(path, []byte(`
cache_dir = "cache"

[[sync]]
app = 10
target = "leafo/a"

[[sync]]
app = 20
target = "leafo/b"
cache_dir = "own"

[[sync]]
app = 30
target = "leafo/c"
cache_dir = "/abs/own"
`), 0o644)

	c, err := LoadSyncConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Find(10).CacheDir != "" {
		t.Fatal("entries should keep their cache dir as written")
	}
	if got := c.Resolve(*c.Find(10)).CacheDir; got != filepath.Join(dir, "cache", "10") {
		t.Fatalf("inherited: %q", got)
	}
	if got := c.Resolve(*c.Find(20)).CacheDir; got != filepath.Join(dir, "own") {
		t.Fatalf("override: %q", got)
	}
	if got := c.Resolve(*c.Find(30)).CacheDir; got != filepath.Join("/abs", "own") {
		t.Fatalf("absolute override: %q", got)
	}

	c.CacheDir = filepath.Join(dir, "elsewhere")
	if got := c.Resolve(*c.Find(10)).CacheDir; got != filepath.Join(dir, "elsewhere", "10") {
		t.Fatalf("absolute global: %q", got)
	}
}
