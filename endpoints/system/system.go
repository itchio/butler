package system

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/httpkit/progress"
	"github.com/pkg/errors"
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
		return nil, errors.WithStack(err)
	}

	consumer := rc.Consumer
	consumer.Statf("(%s): %s free out of %s total",
		params.Path,
		progress.FormatBytes(res.FreeSize),
		progress.FormatBytes(res.TotalSize),
	)
	return res, nil
}
