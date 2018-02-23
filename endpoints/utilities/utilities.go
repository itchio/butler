package utilities

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.VersionGet.Register(router, func(rc *buse.RequestContext, params *buse.VersionGetParams) (*buse.VersionGetResult, error) {
		return &buse.VersionGetResult{
			Version:       rc.MansionContext.Version,
			VersionString: rc.MansionContext.VersionString,
		}, nil
	})
}
