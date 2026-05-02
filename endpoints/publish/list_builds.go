package publish

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func ListBuilds(rc *butlerd.RequestContext, params butlerd.PublishListBuildsParams) (*butlerd.PublishListBuildsResult, error) {
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

	// The API returns games/uploads normalized (one entry each, referenced by
	// id) to avoid response inflation when many builds share a game or upload.
	// Re-attach them here so butlerd consumers see each build with its
	// game/upload nested, which is more convenient for the app to render.
	games := make(map[int64]*itchio.Game, len(res.Games))
	for _, g := range res.Games {
		games[g.ID] = g
	}
	uploads := make(map[int64]*itchio.Upload, len(res.Uploads))
	for _, u := range res.Uploads {
		uploads[u.ID] = u
	}
	for _, b := range res.Builds {
		b.Game = games[b.GameID]
		b.Upload = uploads[b.UploadID]
	}

	out := &butlerd.PublishListBuildsResult{
		Builds:  res.Builds,
		Page:    res.Page,
		PerPage: res.PerPage,
	}
	if res.Totals != nil {
		out.Totals = &butlerd.PublishBuildTotals{
			All:          res.Totals.All,
			Live:         res.Totals.Live,
			Processing:   res.Totals.Processing,
			Failed:       res.Totals.Failed,
			ProjectCount: res.Totals.ProjectCount,
		}
	}

	return out, nil
}
