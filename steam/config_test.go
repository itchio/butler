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
		"missing target": "[[sync]]\napp = 1\n",
		"duplicate app":  "[[sync]]\napp = 1\ntarget = \"a/b\"\n[[sync]]\napp = 1\ntarget = \"a/c\"\n",
		"bad map key":    "[[sync]]\napp = 1\ntarget = \"a/b\"\n[sync.map]\nabc = \"win\"\n",
	} {
		path := filepath.Join(dir, name+".toml")
		os.WriteFile(path, []byte(body), 0o644)
		if _, err := LoadSyncConfig(path); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
