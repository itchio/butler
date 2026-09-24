package push

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itchio/butler/buildinfo"
	"github.com/itchio/butler/filtering"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/dash"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/itchio/lake"
	"github.com/itchio/lake/pools/zippool"
	"github.com/itchio/lake/tlc"
)

func writeLaunchFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeLaunchZip(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("index.html")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(w, "game"); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

type launchTransport func(*http.Request) (*http.Response, error)

func (f launchTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPushSendsLaunchAnalysis(t *testing.T) {
	t.Setenv("BUTLER_API_KEY", "test-key")
	t.Setenv("BUTLER_PUSH_SOURCE", "test-app")
	for _, kind := range []string{"folder", "zip", "auto-unzip", "wrapped-app", "empty"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			src := root
			wantPath := "index.html"
			switch kind {
			case "folder":
				writeLaunchFile(t, root, "index.html", "game")
				writeLaunchFile(t, root, ".itch/index.html", "excluded")
			case "zip", "auto-unzip":
				archive := filepath.Join(root, "game.zip")
				writeLaunchZip(t, archive)
				if kind == "zip" {
					src = archive
				}
			case "wrapped-app":
				src = filepath.Join(root, "Game.app")
				writeLaunchFile(t, src, "index.html", "game")
				writeLaunchFile(t, root, "index.html", "outside upload")
				wantPath = "Game.app/index.html"
			}
			var values url.Values
			ctx := &mansion.Context{ContextTimeout: 10, HTTPClient: &http.Client{Transport: launchTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.Path != "/wharf/builds" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				values = r.PostForm
				// Stop at build creation; this test must never start a real upload.
				return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"errors":["test stops before upload"]}`)), Request: r}, nil
			})}}
			ctx.SetAddress("https://itch.io")
			err := Do(ctx, src, "user/game:linux", "", true, false, false, true, true, false, itchio.BuildMetadata{"steam": map[string]any{"app_id": 123}})
			if err == nil {
				t.Fatal("expected deliberate build creation failure")
			}
			if values == nil {
				t.Fatalf("build request was not sent: %v", err)
			}
			if values.Get("source") != "test-app" || values.Get("metadata") == "" {
				t.Fatalf("lost build metadata: %v", values)
			}
			var report itchio.BuildLaunchAnalysis
			if err := json.Unmarshal([]byte(values.Get("launch_analysis")), &report); err != nil {
				t.Fatal(err)
			}
			if report.SchemaVersion != dash.LaunchTargetsSchemaVersion || !strings.HasPrefix(report.ScannerVersion, "butler/") {
				t.Fatalf("unexpected report: %+v", report)
			}
			var targets []dash.LaunchTarget
			if err := json.Unmarshal(report.LaunchTargets, &targets); err != nil {
				t.Fatal(err)
			}
			if kind == "empty" {
				if report.ExtractedSize != 0 {
					t.Fatalf("empty report size: %d", report.ExtractedSize)
				}
				if string(report.LaunchTargets) != "[]" {
					t.Fatalf("empty report: %s", report.LaunchTargets)
				}
				return
			}
			if report.ExtractedSize != 4 {
				t.Fatalf("unexpected extracted size: %d", report.ExtractedSize)
			}
			if len(targets) != 1 || targets[0].Path != wantPath || targets[0].Size != 4 || targets[0].Sha256 != fmt.Sprintf("%x", sha256.Sum256([]byte("game"))) {
				t.Fatalf("unexpected targets: %s", report.LaunchTargets)
			}
		})
	}
}

func TestLaunchAnalysisUsesSeparatePool(t *testing.T) {
	for _, zipped := range []bool{false, true} {
		t.Run(fmt.Sprintf("zip=%v", zipped), func(t *testing.T) {
			root := t.TempDir()
			src := root
			if zipped {
				src = filepath.Join(root, "game.zip")
				writeLaunchZip(t, src)
			} else {
				writeLaunchFile(t, root, "index.html", "game")
			}
			results := make(chan walkResult, 1)
			errs := make(chan error, 1)
			doWalk(src, results, errs, true, tlc.WalkOpts{Filter: filtering.FilterPaths})
			select {
			case err := <-errs:
				t.Fatal(err)
			case result := <-results:
				defer result.pool.Close()
				if scanBuildLaunchAnalysis(src, result.container, &state.Consumer{}) == nil {
					t.Fatal("scan failed")
				}
				r, err := result.pool.GetReader(0)
				if err != nil {
					t.Fatal(err)
				}
				content, err := io.ReadAll(r)
				if err != nil || string(content) != "game" {
					t.Fatalf("upload read after scan: %q, %v", content, err)
				}
			}
		})
	}
}

func TestLaunchAnalysisClosesZipSpool(t *testing.T) {
	scratch := t.TempDir()
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, scratch)
	}
	archivePath := filepath.Join(t.TempDir(), "game.zip")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("index.html")
	if err != nil {
		t.Fatal(err)
	}
	const size = zippool.DefaultMaxMemory + 1
	if _, err := io.CopyN(w, launchZeros{}, size); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	container, err := tlc.WalkAny(archivePath, tlc.WalkOpts{})
	if err != nil {
		t.Fatal(err)
	}
	report := scanBuildLaunchAnalysis(archivePath, container, &state.Consumer{})
	if report == nil {
		t.Fatal("scan failed")
	}
	var targets []dash.LaunchTarget
	if err := json.Unmarshal(report.LaunchTargets, &targets); err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 || targets[0].Size != size || targets[0].Sha256 == "" {
		t.Fatalf("unexpected report: %s", report.LaunchTargets)
	}
	spools, err := filepath.Glob(filepath.Join(scratch, "lake-zip-*"))
	if err != nil || len(spools) != 0 {
		t.Fatalf("scan left spool files: %v, %v", spools, err)
	}
	if err := os.Remove(archivePath); err != nil {
		t.Fatalf("scan left the archive open: %v", err)
	}
}

type launchZeros struct{}

func (launchZeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

type failingLaunchPool struct{ lake.Pool }

func (failingLaunchPool) GetReadSeeker(int64) (io.ReadSeeker, error) {
	return nil, fmt.Errorf("scan read failed")
}

func TestLaunchAnalysisFailureIsOptional(t *testing.T) {
	warned := false
	consumer := &state.Consumer{OnMessage: func(level, message string) {
		if level == "warning" {
			warned = true
		}
	}}
	report := scanLaunchAnalysis(&tlc.Container{Files: []*tlc.File{{Path: "game.exe", Size: 100}}}, failingLaunchPool{}, consumer)
	if report != nil || !warned {
		t.Fatalf("report=%+v warned=%v", report, warned)
	}
}

func TestLaunchAnalysisScannerVersion(t *testing.T) {
	oldVersion, oldCommit := buildinfo.Version, buildinfo.Commit
	t.Cleanup(func() { buildinfo.Version, buildinfo.Commit = oldVersion, oldCommit })
	for _, tc := range []struct{ version, commit, want string }{
		{"head", "abc123", "butler/abc123"},
		{"1.2.3", "abc123", "butler/1.2.3"},
		{"head", "", "butler/head"},
	} {
		buildinfo.Version, buildinfo.Commit = tc.version, tc.commit
		report := scanLaunchAnalysis(&tlc.Container{}, failingLaunchPool{}, &state.Consumer{})
		if report == nil || report.ScannerVersion != tc.want {
			t.Fatalf("unexpected scanner version: %+v", report)
		}
	}
}

func TestLaunchAnalysisDereferencedSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require privileges on Windows")
	}
	root := t.TempDir()
	writeLaunchFile(t, root, "page.txt", "game")
	if err := os.Symlink("page.txt", filepath.Join(root, "index.html")); err != nil {
		t.Fatal(err)
	}
	results := make(chan walkResult, 1)
	errs := make(chan error, 1)
	doWalk(root, results, errs, true, tlc.WalkOpts{Filter: filtering.FilterPaths, Dereference: true})
	select {
	case err := <-errs:
		t.Fatal(err)
	case result := <-results:
		defer result.pool.Close()
		report := scanLaunchAnalysis(result.container, result.pool, &state.Consumer{})
		if report == nil {
			t.Fatal("scan failed")
		}
		var targets []dash.LaunchTarget
		if err := json.Unmarshal(report.LaunchTargets, &targets); err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 || targets[0].Path != "index.html" || targets[0].Size != 4 {
			t.Fatalf("unexpected targets: %s", report.LaunchTargets)
		}
	}
}
