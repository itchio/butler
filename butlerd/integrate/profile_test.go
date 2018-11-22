package integrate

import (
	"testing"

	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Profile(t *testing.T) {
	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	_, err := messages.ProfileLoginWithAPIKey.TestCall(rc, butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: "meh",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "itch.io API error (403)")

	prof := bi.Authenticate()

	r, err := messages.ProfileList.TestCall(rc, butlerd.ProfileListParams{})
	must(t, err)
	assert.NotEmpty(t, r.Profiles)

	v := uuid.New()

	_, err = messages.ProfileDataPut.TestCall(rc, butlerd.ProfileDataPutParams{
		ProfileID: prof.ID,
		Key:       "@integrate/hello",
		Value:     v.String(),
	})
	must(t, err)

	dgr, err := messages.ProfileDataGet.TestCall(rc, butlerd.ProfileDataGetParams{
		ProfileID: prof.ID,
		Key:       "@integrate/hello",
	})
	must(t, err)
	assert.True(t, dgr.OK)
	assert.EqualValues(t, v.String(), dgr.Value)

	dgr, err = messages.ProfileDataGet.TestCall(rc, butlerd.ProfileDataGetParams{
		ProfileID: prof.ID,
		Key:       "@integrate/whoops",
	})
	must(t, err)
	assert.False(t, dgr.OK)
}
