package tests

import (
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
)

func Register(router *butlerd.Router) {
	messages.TestDoubleTwice.Register(router, func(rc *butlerd.RequestContext, params butlerd.TestDoubleTwiceParams) (*butlerd.TestDoubleTwiceResult, error) {
		if params.Number == 0 {
			return nil, errors.New("number must be non-zero")
		}

		rc.StartProgress()
		consumer := rc.Consumer
		consumer.Progress(0.3)

		number := params.Number
		res, err := messages.TestDouble.Call(rc, butlerd.TestDoubleParams{
			Number: number,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		consumer.Progress(0.6)

		time.Sleep(50 * time.Millisecond)

		consumer.Progress(0.9)

		return &butlerd.TestDoubleTwiceResult{
			Number: res.Number * 2,
		}, nil
	})
}
