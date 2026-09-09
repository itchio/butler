package steam

import (
	"context"

	"github.com/itchio/fresh-steamer/appinfo"
	"github.com/pkg/errors"
)

// AppInfo is PICS app info flattened for reporting. Its job is to show,
// per depot and branch, whether a password branch carries an encrypted
// manifest (supported) or no manifest at all (the newer private-branch
// mechanism, not supported yet), without needing the password or a sync.
type AppInfo struct {
	ID           uint32       `json:"id"`
	Name         string       `json:"name"`
	Type         string       `json:"type,omitempty"`
	ChangeNumber uint32       `json:"change_number"`
	OSList       []string     `json:"oslist,omitempty"`
	OSArch       string       `json:"osarch,omitempty"`
	Branches     []BranchInfo `json:"branches"`
	Depots       []DepotInfo  `json:"depots"`
}

type DepotInfo struct {
	ID            uint32   `json:"id"`
	Name          string   `json:"name,omitempty"`
	OSList        []string `json:"oslist,omitempty"`
	OSArch        string   `json:"osarch,omitempty"`
	Language      string   `json:"language,omitempty"`
	DLCAppID      uint32   `json:"dlc_app_id,omitempty"`
	SharedFromApp uint32   `json:"shared_from_app,omitempty"`
	// Keyed by branch name. A branch missing here has no entry for this
	// depot in app info, which is distinct from an encrypted one.
	Manifests map[string]DepotManifest `json:"manifests"`
}

type DepotManifest struct {
	// GID is zero when Encrypted; the id needs the branch password.
	GID       uint64 `json:"gid,omitempty"`
	Size      uint64 `json:"size,omitempty"`
	Download  uint64 `json:"download,omitempty"`
	Encrypted bool   `json:"encrypted,omitempty"`
}

// GetAppInfo runs the publisher key gate before fetching.
func GetAppInfo(ctx context.Context, s Store, appID uint32) (*AppInfo, error) {
	if err := s.CheckAppAccess(ctx, appID); err != nil {
		return nil, err
	}
	sess, err := s.OpenSession(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer sess.Close()
	app, err := sess.AppInfo(ctx, appID)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching app info for %d", appID)
	}
	return flattenAppInfo(app), nil
}

func flattenAppInfo(app *appinfo.App) *AppInfo {
	out := &AppInfo{
		ID:           app.ID,
		Name:         app.Name,
		Type:         app.Type,
		ChangeNumber: app.ChangeNumber,
		OSList:       app.OSList,
		OSArch:       app.OSArch,
		Branches:     []BranchInfo{},
		Depots:       []DepotInfo{},
	}
	for _, b := range app.Branches {
		out.Branches = append(out.Branches, BranchInfo{
			Name:             b.Name,
			BuildID:          b.BuildID,
			Description:      b.Description,
			PasswordRequired: b.PasswordRequired,
			TimeUpdated:      b.TimeUpdated,
		})
	}
	for _, d := range app.Depots {
		di := DepotInfo{
			ID:            d.ID,
			Name:          d.Name,
			OSList:        d.OSList,
			OSArch:        d.OSArch,
			Language:      d.Language,
			DLCAppID:      d.DLCAppID,
			SharedFromApp: d.SharedFromApp,
			Manifests:     map[string]DepotManifest{},
		}
		for branch, m := range d.Manifests {
			di.Manifests[branch] = DepotManifest{GID: m.GID, Size: m.Size, Download: m.Download}
		}
		for branch := range d.EncryptedManifests {
			di.Manifests[branch] = DepotManifest{Encrypted: true}
		}
		out.Depots = append(out.Depots, di)
	}
	return out
}
