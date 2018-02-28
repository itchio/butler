package tests

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
)

func Register(router *buse.Router) {
	messages.TestDoubleTwice.Register(router, func(rc *buse.RequestContext, params *buse.TestDoubleTwiceParams) (*buse.TestDoubleTwiceResult, error) {
		if params.Number == 0 {
			return nil, errors.New("number must be non-zero")
		}

		res, err := messages.TestDouble.Call(rc, &buse.TestDoubleParams{
			Number: params.Number,
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		return &buse.TestDoubleTwiceResult{
			Number: res.Number * 2,
		}, nil
	})
}
