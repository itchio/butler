package walkutil_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/walkutil"
	"github.com/stretchr/testify/assert"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func TestResolveSingleZipDir_LoneZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "game.zip")
	writeFile(t, zipPath, "PK\x05\x06") // contents irrelevant; helper only checks suffix

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, zipPath, got)
}

func TestResolveSingleZipDir_ZipWithFilteredCruft(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "game.zip")
	writeFile(t, zipPath, "x")
	writeFile(t, filepath.Join(dir, ".DS_Store"), "x")
	writeFile(t, filepath.Join(dir, "Thumbs.db"), "x")

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, zipPath, got, "filtered cruft should not block unwrap")
}

func TestResolveSingleZipDir_UpperCaseExtension(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "GAME.ZIP")
	writeFile(t, zipPath, "x")

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, zipPath, got, "suffix check should be case-insensitive")
}

func TestResolveSingleZipDir_TwoEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game.zip"), "x")
	writeFile(t, filepath.Join(dir, "readme.txt"), "x")

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, dir, got, "two non-filtered entries should not unwrap")
}

func TestResolveSingleZipDir_SubdirOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, dir, got, "single subdirectory should not unwrap")
}

func TestResolveSingleZipDir_SubdirNamedZip(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "weird.zip"), 0755); err != nil {
		t.Fatal(err)
	}

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, dir, got, "directory named *.zip should not unwrap (not a regular file)")
}

func TestResolveSingleZipDir_SymlinkToZip(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated permissions on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "real.zip")
	writeFile(t, target, "x")
	if err := os.Symlink(target, filepath.Join(dir, "link.zip")); err != nil {
		t.Fatal(err)
	}

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, dir, got, "symlink to zip should not unwrap (avoids escaping the directory)")
}

func TestResolveSingleZipDir_NonZipExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game.zip data"), "x")

	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, dir, got, "filename with trailing chars after .zip should not unwrap")
}

func TestResolveSingleZipDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	got := walkutil.ResolveSingleZipDir(dir, filtering.FilterPaths)
	assert.Equal(t, dir, got)
}

func TestResolveSingleZipDir_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "game.zip")
	writeFile(t, filePath, "x")

	got := walkutil.ResolveSingleZipDir(filePath, filtering.FilterPaths)
	assert.Equal(t, filePath, got, "passing a file directly should be a no-op")
}

func TestResolveSingleZipDir_NonExistent(t *testing.T) {
	got := walkutil.ResolveSingleZipDir("/this/path/does/not/exist", filtering.FilterPaths)
	assert.Equal(t, "/this/path/does/not/exist", got)
}
