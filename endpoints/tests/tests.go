package tests

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
)

func Register(router *buse.Router) {
	router.Register("Test.DoubleTwice", func(rc *buse.RequestContext) (interface{}, error) {
		var ddreq buse.TestDoubleTwiceParams

		return rc.WithParams(&ddreq, func() (interface{}, error) {
			var dres buse.TestDoubleResult
			err := rc.Call("Test.Double", &buse.TestDoubleParams{Number: ddreq.Number}, &dres)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			return &buse.TestDoubleTwiceResult{
				Number: dres.Number * 2,
			}, nil
		})
	})
}
