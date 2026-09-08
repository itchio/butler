package steam

import (
	"context"
	"os"
	"sort"

	"github.com/itchio/fresh-steamer/appinfo"
	"github.com/itchio/fresh-steamer/partner"
)

// EnvUngated turns off the publisher key gate for development: any key is
// accepted without asking Steam, ListApps returns the apps the logged-in
// account owns, and CheckAppAccess passes. It exists so the whole flow
// can be exercised without a partner account. Callers should say so
// loudly whenever it is on.
const EnvUngated = "BUTLER_STEAM_UNGATED"

func Ungated() bool {
	return os.Getenv(EnvUngated) == "1"
}

// OwnedApps lists the apps the stored Steam login holds a license for.
// Package 0, the free tools every account has, is left out.
func OwnedApps(ctx context.Context, s Store) ([]partner.App, error) {
	sess, err := s.OpenSession(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	licenses, err := appinfo.Licenses(ctx, sess.CM)
	if err != nil {
		return nil, err
	}
	packages, err := appinfo.Packages(ctx, sess.CM, licenses)
	if err != nil {
		return nil, err
	}
	seen := map[uint32]bool{}
	var ids []uint32
	for _, p := range packages {
		if p.ID == 0 {
			continue
		}
		for _, a := range p.AppIDs {
			if !seen[a] {
				seen[a] = true
				ids = append(ids, a)
			}
		}
	}
	names, err := appinfo.Names(ctx, sess.CM, ids)
	if err != nil {
		return nil, err
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	apps := make([]partner.App, 0, len(ids))
	for _, id := range ids {
		apps = append(apps, partner.App{ID: id, Name: names[id]})
	}
	return apps, nil
}
