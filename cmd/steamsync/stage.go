package steamsync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/fresh-steamer/depot"
	"github.com/itchio/fresh-steamer/session"
	"github.com/itchio/headway/united"
	"github.com/pkg/errors"
)

// Staging layout under the sync directory:
//
//	depots/<depot id>/    one download per depot, reused across syncs
//	store/                manifests and resume journals for those downloads
//	channels/<name>/      what gets pushed, hardlinked from depots/
//
// Depots are downloaded once even when several channels share them. With a
// persistent --cache-dir, unchanged files are skipped on the next sync and
// an interrupted download resumes.
type stage struct {
	Dir string
}

// openStage returns the staging directory for this run. With cacheDir set
// it is used as is and kept. Otherwise a fresh directory is created under
// butler's own data directory rather than the system temp dir, which on
// Linux is often RAM-backed, and the returned cleanup removes it.
func openStage(ctx *mansion.Context, cacheDir string) (*stage, func(), error) {
	if cacheDir != "" {
		return &stage{Dir: cacheDir}, func() {}, nil
	}
	base := filepath.Join(filepath.Dir(ctx.Identity), "steam-sync")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, nil, errors.Wrapf(err, "creating %s", base)
	}
	sweepStale(base)
	dir, err := os.MkdirTemp(base, "tmp-")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "creating temporary directory in %s", base)
	}
	comm.Debugf("Staging in %s (pass --cache-dir to keep downloads between syncs)", dir)
	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			comm.Warnf("Could not remove %s: %v", dir, err)
		}
	}
	return &stage{Dir: dir}, cleanup, nil
}

// sweepStale removes temporary staging directories left behind by runs
// that were killed before cleanup. Anything touched in the last day is
// assumed to belong to a sync still in progress.
func sweepStale(base string) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "tmp-") {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < 24*time.Hour {
			continue
		}
		p := filepath.Join(base, e.Name())
		comm.Debugf("Removing stale staging directory %s", p)
		if err := os.RemoveAll(p); err != nil {
			comm.Warnf("Could not remove %s: %v", p, err)
		}
	}
}

func (st *stage) depotDir(id uint32) string {
	return filepath.Join(st.Dir, "depots", strconv.FormatUint(uint64(id), 10))
}

func (st *stage) channelDir(name string) string {
	return filepath.Join(st.Dir, "channels", name)
}

func (st *stage) store() *depot.Store {
	return &depot.Store{Dir: filepath.Join(st.Dir, "store")}
}

// downloadDepot brings depots/<id> up to date with the planned manifest.
func (st *stage) downloadDepot(goCtx context.Context, s *session.Session, plan *Plan, dp *DepotPlan, password string) error {
	key, err := s.DepotKey(goCtx, plan.AppID, dp.ID)
	if err != nil {
		return errors.Wrapf(err, "getting decryption key for depot %d (Steam refuses keys for some unreleased apps unless the account owns them)", dp.ID)
	}
	code, err := s.ManifestRequestCode(goCtx, plan.AppID, dp.ID, dp.GID, plan.Branch, password)
	if err != nil {
		return errors.Wrapf(err, "requesting manifest access for depot %d", dp.ID)
	}
	cdnClient, err := s.CDN(goCtx)
	if err != nil {
		return errors.Wrap(err, "picking Steam content servers")
	}
	manifest, err := cdnClient.FetchManifest(goCtx, dp.ID, dp.GID, code, key)
	if err != nil {
		return errors.Wrapf(err, "fetching manifest for depot %d", dp.ID)
	}

	comm.Opf("Downloading depot %d (%d files)", dp.ID, len(manifest.Files))
	comm.StartProgressWithTotalBytes(int64(manifest.TotalSize))
	var last depot.Progress
	err = depot.Download(goCtx, cdnClient, depot.Options{
		Dir:      st.depotDir(dp.ID),
		DepotID:  dp.ID,
		DepotKey: key,
		Manifest: manifest,
		Store:    st.store(),
		Logf:     comm.Debugf,
		OnProgress: func(p depot.Progress) {
			last = p
			if p.BytesTotal > 0 {
				comm.Progress(float64(p.BytesDone) / float64(p.BytesTotal))
			}
		},
	})
	comm.EndProgress()
	if err != nil {
		return errors.Wrapf(err, "downloading depot %d", dp.ID)
	}
	fetched := last.BytesTotal - last.BytesSkipped - last.BytesReused
	comm.Statf("Depot %d: %s fetched, %s reused from previous files, %s unchanged",
		dp.ID, united.FormatBytes(int64(fetched)), united.FormatBytes(int64(last.BytesReused)), united.FormatBytes(int64(last.BytesSkipped)))
	return nil
}

// assembleChannel rebuilds channels/<name> from its depots. Files are
// hardlinked so a multi-depot channel costs no extra disk, with a copy
// as fallback when the filesystem refuses.
func (st *stage) assembleChannel(c *ChannelPlan) (string, error) {
	dir := st.channelDir(c.Name)
	if err := os.RemoveAll(dir); err != nil {
		return "", errors.Wrapf(err, "clearing %s", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, dp := range c.Depots {
		src := st.depotDir(dp.ID)
		err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			dst := filepath.Join(dir, rel)
			info, err := d.Info()
			if err != nil {
				return err
			}
			switch {
			case d.IsDir():
				return os.MkdirAll(dst, 0o755)
			case info.Mode()&os.ModeSymlink != 0:
				target, err := os.Readlink(path)
				if err != nil {
					return err
				}
				os.Remove(dst)
				return os.Symlink(target, dst)
			default:
				return link(path, dst)
			}
		})
		if err != nil {
			return "", errors.Wrapf(err, "assembling channel %s from depot %d", c.Name, dp.ID)
		}
	}
	return dir, nil
}

func link(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	// Later depots win when two ship the same path, which matches how
	// Steam mounts them.
	os.Remove(dst)
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// steamworksFiles lists files in dir that mean the build talks to the
// Steam client. Such a build may refuse to start outside Steam.
func steamworksFiles(dir string) []string {
	var found []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		switch {
		case strings.HasPrefix(name, "steam_api") && (strings.HasSuffix(name, ".dll") || strings.HasSuffix(name, ".lib")),
			name == "libsteam_api.so", name == "libsteam_api.dylib",
			name == "steam_appid.txt",
			strings.HasPrefix(name, "steamclient") && strings.HasSuffix(name, ".dll"):
			rel, _ := filepath.Rel(dir, path)
			found = append(found, rel)
		}
		return nil
	})
	return found
}

func warnSteamworks(channel string, files []string) {
	if len(files) == 0 {
		return
	}
	lines := []string{
		fmt.Sprintf("The %s build ships the Steamworks SDK:", channel),
		"",
	}
	for _, f := range files {
		lines = append(lines, "  "+f)
	}
	lines = append(lines, "",
		"If the game initializes Steam at startup it may not run for itch.io players.",
		"Consider a build with Steam integration disabled for this channel.")
	comm.Notice("Steamworks SDK detected", lines)
}
