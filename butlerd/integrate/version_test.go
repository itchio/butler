package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Version(t *testing.T) {
	rc, _, cancel := newInstance(t).Unwrap()
	defer cancel()

	vgr, err := messages.VersionGet.TestCall(rc, butlerd.VersionGetParams{})
	must(t, err)
	assert.NotEmpty(t, vgr.Version)
}
