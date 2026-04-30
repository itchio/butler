package wharf

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func ListBuilds(rc *butlerd.RequestContext, params butlerd.WharfListBuildsParams) (*butlerd.WharfListBuildsResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.ListProfileBuilds(rc.Ctx, itchio.ListProfileBuildsParams{
		Page:          params.Page,
		PerPage:       params.PerPage,
		State:         params.State,
		IncludeTotals: params.IncludeTotals,
	})
	if err != nil {
		return nil, errors.Wrap(err, "listing profile builds")
	}

	out := &butlerd.WharfListBuildsResult{
		Builds:  res.Builds,
		Page:    res.Page,
		PerPage: res.PerPage,
	}
	if res.Totals != nil {
		out.Totals = &butlerd.WharfBuildTotals{
			All:          res.Totals.All,
			Live:         res.Totals.Live,
			Processing:   res.Totals.Processing,
			Failed:       res.Totals.Failed,
			ProjectCount: res.Totals.ProjectCount,
		}
	}

	return out, nil
}
