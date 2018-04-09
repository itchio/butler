package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Version(t *testing.T) {
	rc, _, cancel := connect(t)
	defer cancel()

	vgr, err := messages.VersionGet.TestCall(rc, nil)
	must(t, err)
	assert.EqualValues(t, vgr.Version, "head")
}
