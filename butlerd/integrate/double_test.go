package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Double(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	messages.TestDouble.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.TestDoubleParams) (*butlerd.TestDoubleResult, error) {
		return &butlerd.TestDoubleResult{
			Number: params.Number * 2,
		}, nil
	})

	res, err := messages.TestDoubleTwice.TestCall(rc, butlerd.TestDoubleTwiceParams{Number: 512})
	must(t, err)
	assert.EqualValues(t, 2048, res.Number)
}
