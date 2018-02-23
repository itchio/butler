package utilities

import (
	"github.com/itchio/butler/buse"
)

func Register(router *buse.Router) {
	router.Register("Version.Get", func(rc *buse.RequestContext) (interface{}, error) {
		res := &buse.VersionGetResult{
			Version:       rc.MansionContext.Version,
			VersionString: rc.MansionContext.VersionString,
		}
		return res, nil
	})
}
