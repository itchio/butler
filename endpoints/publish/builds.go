package publish

import (
	"github.com/itchio/butler/butlerd"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func GetBuild(rc *butlerd.RequestContext, params butlerd.PublishGetBuildParams) (*butlerd.PublishGetBuildResult, error) {
	_, client := rc.ProfileClient(params.ProfileID)

	res, err := client.GetBuild(rc.Ctx, itchio.GetBuildParams{
		BuildID: params.BuildID,
	})
	if err != nil {
		return nil, errors.Wrap(err, "getting build")
	}

	return &butlerd.PublishGetBuildResult{
		Build: res.Build,
	}, nil
}
