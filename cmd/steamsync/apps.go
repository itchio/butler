package steamsync

import (
	"context"
	"fmt"
	"sort"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/fresh-steamer/appinfo"
	"github.com/pkg/errors"
)

var appsArgs = struct {
	owned bool
}{}

func RegisterApps(ctx *mansion.Context) {
	cmd := ctx.App.Command("steam-apps", "List the Steam apps your publisher key controls.").Hidden()
	cmd.Flag("owned", "List apps the logged-in Steam account holds a license for instead. Those cannot be synced unless the publisher key also controls them.").BoolVar(&appsArgs.owned)
	ctx.Register(cmd, doApps)
}

func doApps(ctx *mansion.Context) {
	if appsArgs.owned {
		ctx.Must(OwnedApps(ctx))
	} else {
		ctx.Must(Apps(ctx))
	}
}

type appRow struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	Via  string `json:"via,omitempty"`
}

func Apps(ctx *mansion.Context) error {
	goCtx, cancel := ctx.DefaultCtx()
	defer cancel()

	pc, err := partnerClient(ctx)
	if err != nil {
		return err
	}
	apps, err := pc.Apps(goCtx)
	if err != nil {
		return errors.Wrap(err, "listing partner apps")
	}
	rows := make([]appRow, 0, len(apps))
	for _, a := range apps {
		rows = append(rows, appRow{ID: a.ID, Name: a.Name, Type: a.Type})
	}
	printApps(rows, "type")
	return nil
}

func OwnedApps(ctx *mansion.Context) error {
	goCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := openSession(ctx, goCtx)
	if err != nil {
		return err
	}
	defer s.Close()

	licenses, err := appinfo.Licenses(goCtx, s.CM)
	if err != nil {
		return errors.Wrap(err, "listing licenses")
	}
	packages, err := appinfo.Packages(goCtx, s.CM, licenses)
	if err != nil {
		return errors.Wrap(err, "fetching packages")
	}
	via := map[uint32][]string{}
	var ids []uint32
	for _, p := range packages {
		// package 0 is the free tools every account has
		if p.ID == 0 {
			continue
		}
		for _, a := range p.AppIDs {
			if _, seen := via[a]; !seen {
				ids = append(ids, a)
			}
			label := fmt.Sprintf("%d (%s)", p.ID, appinfo.BillingTypeName(p.BillingType))
			if p.DeveloperOnly {
				label += " [developer]"
			}
			via[a] = append(via[a], label)
		}
	}
	names, err := appinfo.Names(goCtx, s.CM, ids)
	if err != nil {
		return errors.Wrap(err, "fetching app names")
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	rows := make([]appRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, appRow{ID: id, Name: names[id], Via: joinVia(via[id])})
	}
	printApps(rows, "via package")
	return nil
}

func joinVia(labels []string) string {
	out := ""
	for i, l := range labels {
		if i > 0 {
			out += "; "
		}
		out += l
	}
	return out
}

func printApps(rows []appRow, extraHeader string) {
	comm.ResultOrPrint(rows, func() {
		if len(rows) == 0 {
			comm.Log("No apps found.")
			return
		}
		comm.Logf("%-10s %-40s %s", "app id", "name", extraHeader)
		for _, r := range rows {
			extra := r.Type
			if extra == "" {
				extra = r.Via
			}
			comm.Logf("%-10d %-40s %s", r.ID, r.Name, extra)
		}
	})
}
