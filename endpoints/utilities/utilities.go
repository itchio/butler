package utilities

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.VersionGet.Register(router, func(rc *butlerd.RequestContext, params *butlerd.VersionGetParams) (*butlerd.VersionGetResult, error) {
		return &butlerd.VersionGetResult{
			Version:       rc.ButlerVersion,
			VersionString: rc.ButlerVersionString,
		}, nil
	})
}
