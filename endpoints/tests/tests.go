package tests

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
)

func Register(router *butlerd.Router) {
	messages.TestDoubleTwice.Register(router, func(rc *butlerd.RequestContext, params butlerd.TestDoubleTwiceParams) (*butlerd.TestDoubleTwiceResult, error) {
		if params.Number == 0 {
			return nil, errors.New("number must be non-zero")
		}

		res, err := messages.TestDouble.Call(rc, butlerd.TestDoubleParams{
			Number: params.Number,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return &butlerd.TestDoubleTwiceResult{
			Number: res.Number * 2,
		}, nil
	})
}
