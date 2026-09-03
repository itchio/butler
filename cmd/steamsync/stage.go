package steamsync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/fresh-steamer/depot"
	"github.com/itchio/fresh-steamer/session"
	"github.com/itchio/headway/united"
	"github.com/pkg/errors"
)

// Staging layout under the sync directory for one app:
//
//	depots/<depot id>/    one download per depot, reused across syncs
//	store/                manifests and resume journals for those downloads
//	channels/<name>/      what gets pushed, hardlinked from depots/
//
// Depots are downloaded once even when several channels share them, and
// unchanged files are skipped on the next sync.
type stage struct {
	Dir string
}

func defaultStageDir(ctx *mansion.Context, appID uint32) string {
	return filepath.Join(filepath.Dir(ctx.Identity), "steam-sync", strconv.FormatUint(uint64(appID), 10))
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
