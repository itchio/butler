package launch

import (
	"path/filepath"
	"strings"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func GetTargets(rc *butlerd.RequestContext, params butlerd.LaunchGetTargetsParams) (*butlerd.LaunchGetTargetsResult, error) {
	// Deliberately not taking the runlock: Launch holds it for the entire
	// game session, so locking here would block a client's target/settings
	// UI until the game exits. Discovery only reads the install folder, and
	// Launch re-derives targets under its own lock, so a snapshot taken
	// during an install operation can at worst fail to match later.
	info, err := resolveInstallFolderInfo(rc, params.CaveID)
	if err != nil {
		return nil, err
	}

	hosts, err := rc.HostEnumerator().Enumerate(rc.Consumer)
	if err != nil {
		return nil, err
	}

	targetRes, err := getTargets(rc, getTargetsParams{
		info:  info,
		hosts: hosts,
	})
	if err != nil {
		return nil, err
	}

	return &butlerd.LaunchGetTargetsResult{
		Targets: targetRes.targets,
	}, nil
}

// settingsLaunchTarget returns the launch target persisted in the cave's
// settings, or empty if unset or unreadable.
func settingsLaunchTarget(rc *butlerd.RequestContext, cave *models.Cave) string {
	var settings butlerd.CaveSettings
	err := models.UnmarshalJSONAllowEmpty(cave.Settings, &settings, "cave settings")
	if err != nil {
		rc.Consumer.Warnf("Could not parse cave settings: %v", err)
		return ""
	}
	return settings.LaunchTarget
}

// findTarget matches a preferred target against action names first,
// then against action paths (slash-normalized, relative to the install
// folder). Targets are in host preference order, so the first match is
// the one for the most preferred host. Returns nil when nothing matches.
func findTarget(targets []*butlerd.LaunchTarget, preferred string) *butlerd.LaunchTarget {
	for _, t := range targets {
		if t.Action != nil && t.Action.Name == preferred {
			return t
		}
	}

	normalize := func(p string) string {
		return strings.TrimPrefix(filepath.ToSlash(p), "./")
	}
	want := normalize(preferred)
	for _, t := range targets {
		if t.Action != nil && t.Action.Path != "" && normalize(t.Action.Path) == want {
			return t
		}
	}
	return nil
}
