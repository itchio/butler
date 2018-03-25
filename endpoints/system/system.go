package system

import (
	humanize "github.com/dustin/go-humanize"
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Register(router *butlerd.Router) {
	messages.SystemStatFS.Register(router, StatFSHandler)
}

func StatFSHandler(rc *butlerd.RequestContext, params *butlerd.SystemStatFSParams) (*butlerd.SystemStatFSResult, error) {
	if params.Path == "" {
		return nil, errors.Errorf("path must be set")
	}

	res, err := StatFS(params.Path)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	consumer := rc.Consumer
	consumer.Statf("(%s): %s free out of %s total",
		params.Path,
		humanize.IBytes(uint64(res.FreeSize)),
		humanize.IBytes(uint64(res.TotalSize)),
	)
	return res, nil
}
