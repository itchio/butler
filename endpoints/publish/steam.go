package publish

import (
	"errors"
	"strconv"
	"sync"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/steam"
	"github.com/itchio/fresh-steamer/partner"
	"github.com/itchio/fresh-steamer/webapi"
)

func registerSteam(router *butlerd.Router) {
	messages.PublishSteamSyncGetStatus.Register(router, SteamGetStatus)
	messages.PublishSteamSyncLogin.Register(router, SteamLogin)
	messages.PublishSteamSyncLoginCancel.Register(router, SteamLoginCancel)
	messages.PublishSteamSyncLogout.Register(router, SteamLogout)
	messages.PublishSteamSyncSetPublisherKey.Register(router, SteamSetPublisherKey)
	messages.PublishSteamSyncRemovePublisherKey.Register(router, SteamRemovePublisherKey)
	messages.PublishSteamSyncListApps.Register(router, SteamListApps)
	messages.PublishSteamSyncPlan.Register(router, SteamPlan)
	messages.PublishSteamSyncSync.Register(router, SteamSync)
	messages.PublishSteamSyncCancel.Register(router, SteamSyncCancel)
}

func steamStore(rc *butlerd.RequestContext) steam.Store {
	if steam.Ungated() {
		rc.Consumer.Warnf("%s is set: the publisher key is not checked and any owned app can be synced. Development only.", steam.EnvUngated)
	}
	return steam.StoreFor(rc.Identity)
}

func mapSteamErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, steam.ErrNotLoggedIn):
		return butlerd.CodePublishSteamSyncNotLoggedIn
	case errors.Is(err, steam.ErrNoPublisherKey):
		return butlerd.CodePublishSteamSyncNoPublisherKey
	case errors.Is(err, steam.ErrPublisherKeyInvalid):
		return butlerd.CodePublishSteamSyncPublisherKeyInvalid
	}
	return err
}

func steamPlanOptions(appID int64, target, branch, password string, m map[string]string, skip []int64) (steam.PlanOptions, error) {
	entry := steam.SyncEntry{
		App:    uint32(appID),
		Target: target,
		Branch: branch,
		Map:    m,
	}
	for _, id := range skip {
		entry.Skip = append(entry.Skip, uint32(id))
	}
	if err := entry.Validate(); err != nil {
		return steam.PlanOptions{}, err
	}
	return entry.PlanOptions(password)
}

func SteamGetStatus(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncGetStatusParams) (*butlerd.PublishSteamSyncGetStatusResult, error) {
	c, err := steamStore(rc).Load()
	if err != nil {
		return nil, err
	}
	res := &butlerd.PublishSteamSyncGetStatusResult{
		LoggedIn:        c.LoggedIn(),
		HasPublisherKey: c.HasPublisherKey(),
	}
	if c.LoggedIn() {
		res.AccountName = c.AccountName
		res.SteamID = strconv.FormatUint(c.SteamID, 10)
	}
	return res, nil
}

// One QR session at a time, so the app never has two challenges alive.
var steamLoginMu sync.Mutex

func SteamLogin(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncLoginParams) (*butlerd.PublishSteamSyncLoginResult, error) {
	if !steamLoginMu.TryLock() {
		return nil, butlerd.CodePublishSteamSyncLoginInProgress
	}
	defer steamLoginMu.Unlock()

	ctx, cleanup := rc.MakeCancelable(params.ID)
	defer cleanup()

	acct, err := steam.LoginQR(ctx, steamStore(rc), func(url string) {
		_ = messages.PublishSteamSyncLoginChallenge.Notify(rc, butlerd.PublishSteamSyncLoginChallengeNotification{
			ID:  params.ID,
			URL: url,
		})
	}, steam.LoginOptions{Persist: true})
	if err != nil {
		if ctx.Err() != nil {
			return nil, butlerd.CodeOperationCancelled
		}
		if isSteamLoginDenied(err) {
			return nil, butlerd.CodePublishSteamSyncLoginDenied
		}
		return nil, err
	}
	return &butlerd.PublishSteamSyncLoginResult{
		AccountName: acct.AccountName,
		SteamID:     strconv.FormatUint(acct.SteamID, 10),
	}, nil
}

// A declined or expired QR challenge comes back from the poll as a Steam
// result code: FileNotFound (9) once the session is gone, Expired (27),
// or AccessDenied (15).
func isSteamLoginDenied(err error) bool {
	var we *webapi.Error
	if !errors.As(err, &we) {
		return false
	}
	switch we.EResult {
	case 9, 15, 27:
		return true
	}
	return false
}

func SteamLoginCancel(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncLoginCancelParams) (*butlerd.PublishSteamSyncLoginCancelResult, error) {
	return &butlerd.PublishSteamSyncLoginCancelResult{DidCancel: rc.CancelFuncs.Call(params.ID)}, nil
}

func SteamLogout(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncLogoutParams) (*butlerd.PublishSteamSyncLogoutResult, error) {
	if err := steamStore(rc).Logout(); err != nil {
		return nil, err
	}
	return &butlerd.PublishSteamSyncLogoutResult{}, nil
}

func SteamSetPublisherKey(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncSetPublisherKeyParams) (*butlerd.PublishSteamSyncSetPublisherKeyResult, error) {
	apps, err := steam.SetPublisherKey(rc.Ctx, steamStore(rc), params.Key)
	if err != nil {
		return nil, mapSteamErr(err)
	}
	return &butlerd.PublishSteamSyncSetPublisherKeyResult{AppCount: int64(len(apps))}, nil
}

func SteamRemovePublisherKey(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncRemovePublisherKeyParams) (*butlerd.PublishSteamSyncRemovePublisherKeyResult, error) {
	if err := steam.RemovePublisherKey(steamStore(rc)); err != nil {
		return nil, err
	}
	return &butlerd.PublishSteamSyncRemovePublisherKeyResult{}, nil
}

func SteamListApps(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncListAppsParams) (*butlerd.PublishSteamSyncListAppsResult, error) {
	apps, err := steam.ListApps(rc.Ctx, steamStore(rc))
	if err != nil {
		return nil, mapSteamErr(err)
	}
	return &butlerd.PublishSteamSyncListAppsResult{Apps: convertSteamApps(apps)}, nil
}

func convertSteamApps(apps []partner.App) []*butlerd.PublishSteamSyncApp {
	out := make([]*butlerd.PublishSteamSyncApp, 0, len(apps))
	for _, a := range apps {
		out = append(out, &butlerd.PublishSteamSyncApp{ID: int64(a.ID), Name: a.Name, Type: a.Type})
	}
	return out
}

func SteamPlan(rc *butlerd.RequestContext, params butlerd.PublishSteamSyncPlanParams) (*butlerd.PublishSteamSyncPlanResult, error) {
	opts, err := steamPlanOptions(params.AppID, params.Target, params.Branch, params.Password, params.Map, params.Skip)
	if err != nil {
		return nil, err
	}
	plan, err := steam.Plan(rc.Ctx, steamStore(rc), opts)
	if err != nil {
		return nil, mapSteamErr(err)
	}
	return &butlerd.PublishSteamSyncPlanResult{Plan: convertSteamPlan(plan)}, nil
}

func convertSteamPlan(p *steam.SyncPlan) *butlerd.PublishSteamSyncPlan {
	out := &butlerd.PublishSteamSyncPlan{
		AppID:    int64(p.AppID),
		AppName:  p.AppName,
		Branch:   p.Branch,
		BuildID:  int64(p.BuildID),
		Target:   p.Target,
		Channels: []*butlerd.PublishSteamSyncChannel{},
		Skipped:  []*butlerd.PublishSteamSyncSkippedDepot{},
		Warnings: p.Warnings,
		Branches: []*butlerd.PublishSteamSyncBranch{},
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	for _, c := range p.Channels {
		size, download := c.Size()
		ch := &butlerd.PublishSteamSyncChannel{
			Name:     c.Name,
			OS:       c.OS,
			Arch:     c.Arch,
			Depots:   []*butlerd.PublishSteamSyncDepot{},
			Size:     int64(size),
			Download: int64(download),
		}
		for _, d := range c.Depots {
			ch.Depots = append(ch.Depots, &butlerd.PublishSteamSyncDepot{
				ID:       int64(d.ID),
				Name:     d.Name,
				Manifest: strconv.FormatUint(d.GID, 10),
				Size:     int64(d.Size),
				Download: int64(d.Download),
				Shared:   d.Shared,
			})
		}
		out.Channels = append(out.Channels, ch)
	}
	for _, d := range p.Skipped {
		out.Skipped = append(out.Skipped, &butlerd.PublishSteamSyncSkippedDepot{ID: int64(d.ID), Name: d.Name, Reason: d.Reason})
	}
	for _, b := range p.Branches {
		out.Branches = append(out.Branches, &butlerd.PublishSteamSyncBranch{
			Name:             b.Name,
			BuildID:          int64(b.BuildID),
			Description:      b.Description,
			PasswordRequired: b.PasswordRequired,
			TimeUpdated:      int64(b.TimeUpdated),
		})
	}
	return out
}
