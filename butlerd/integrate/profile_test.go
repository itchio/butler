package integrate

import (
	"testing"

	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Profile(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	_, err := messages.ProfileLoginWithAPIKey.TestCall(rc, butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: "meh",
	})
	assert.Error(err)
	assert.Contains(err.Error(), "itch.io API error (403)")

	prof := bi.Authenticate()

	r, err := messages.ProfileList.TestCall(rc, butlerd.ProfileListParams{})
	must(err)
	assert.NotEmpty(r.Profiles)

	v := uuid.New()

	_, err = messages.ProfileDataPut.TestCall(rc, butlerd.ProfileDataPutParams{
		ProfileID: prof.ID,
		Key:       "@integrate/hello",
		Value:     v.String(),
	})
	must(err)

	dgr, err := messages.ProfileDataGet.TestCall(rc, butlerd.ProfileDataGetParams{
		ProfileID: prof.ID,
		Key:       "@integrate/hello",
	})
	must(err)
	assert.True(dgr.OK)
	assert.EqualValues(v.String(), dgr.Value)

	dgr, err = messages.ProfileDataGet.TestCall(rc, butlerd.ProfileDataGetParams{
		ProfileID: prof.ID,
		Key:       "@integrate/whoops",
	})
	must(err)
	assert.False(dgr.OK)
}
