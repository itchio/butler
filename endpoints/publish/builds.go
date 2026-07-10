package publish

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func GetBuild(rc *butlerd.RequestContext, params butlerd.PublishGetBuildParams) (*butlerd.PublishGetBuildResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.GetWharfBuild(rc.Ctx, itchio.GetWharfBuildParams{
		BuildID: params.BuildID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting build")
	}
	if res.Build == nil {
		return nil, errors.Errorf("API returned no build for id %d", params.BuildID)
	}

	return &butlerd.PublishGetBuildResult{
		Build: res.Build,
	}, nil
}
