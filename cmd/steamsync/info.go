package steamsync

import (
	"fmt"
	"strings"

	"github.com/itchio/butler/comm"
	"github.com/itchio/butler/mansion"
	"github.com/itchio/butler/steam"
)

var infoArgs = struct {
	appID uint32
}{}

func RegisterInfo(ctx *mansion.Context) {
	cmd := ctx.App.Command("steam-info", "Show how Steam describes an app: branches, depots, and which branches each depot has a manifest for. Nothing is downloaded.").Hidden()
	cmd.Arg("appid", "Steam app id").Required().Uint32Var(&infoArgs.appID)
	registerCredFlags(cmd)
	ctx.Register(cmd, doInfo)
}

func doInfo(ctx *mansion.Context) {
	ctx.Must(Info(ctx))
}

func Info(ctx *mansion.Context) error {
	goCtx, cancel := ctx.DefaultCtx()
	defer cancel()

	warnUngated()
	comm.Opf("Fetching Steam app info for %d", infoArgs.appID)
	info, err := steam.GetAppInfo(goCtx, store(ctx), infoArgs.appID)
	if err != nil {
		return hint(err)
	}
	comm.ResultOrPrint(info, func() { printInfo(info) })
	return nil
}

func printInfo(info *steam.AppInfo) {
	comm.Logf("")
	comm.Statf("%s (app %d), type %s, change %d", info.Name, info.ID, orDash(info.Type), info.ChangeNumber)
	comm.Logf("  oslist %s, osarch %s", orDash(strings.Join(info.OSList, ",")), orDash(info.OSArch))

	comm.Logf("")
	comm.Logf("branches:")
	for _, b := range info.Branches {
		extra := ""
		if b.PasswordRequired {
			extra = "  password required"
		}
		if b.Description != "" {
			extra += "  " + b.Description
		}
		comm.Logf("  %-20s build %-10d%s", b.Name, b.BuildID, extra)
	}

	comm.Logf("")
	comm.Logf("depots:")
	for _, d := range info.Depots {
		attrs := []string{}
		if len(d.OSList) > 0 {
			attrs = append(attrs, strings.Join(d.OSList, ","))
		}
		if d.OSArch != "" {
			attrs = append(attrs, "arch "+d.OSArch)
		}
		if d.Language != "" {
			attrs = append(attrs, "lang "+d.Language)
		}
		if d.DLCAppID != 0 {
			attrs = append(attrs, fmt.Sprintf("dlc %d", d.DLCAppID))
		}
		if d.SharedFromApp != 0 {
			attrs = append(attrs, fmt.Sprintf("shared from %d", d.SharedFromApp))
		}
		comm.Logf("  %-10d %-30q %s", d.ID, d.Name, strings.Join(attrs, ", "))
		for _, b := range info.Branches {
			m, ok := d.Manifests[b.Name]
			var state string
			switch {
			case ok && m.Encrypted:
				state = "encrypted (older password mechanism, supported)"
			case ok:
				state = fmt.Sprintf("manifest %d", m.GID)
			case b.PasswordRequired:
				state = "missing (private branch, depot section withheld, not yet supported)"
			default:
				state = "missing"
			}
			comm.Logf("    %-20s %s", b.Name, state)
		}
	}
	comm.Logf("")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
