package integrate

import (
	"os"
	"testing"

	"github.com/itchio/butler/butlerd"
	uuid "github.com/satori/go.uuid"

	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Profile(t *testing.T) {
	rc, _, cancel := connect(t)
	defer cancel()

	_, err := messages.ProfileLoginWithAPIKey.TestCall(rc, &butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: "meh",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403 Forbidden")

	prof := authenticate(t, rc)

	r, err := messages.ProfileList.TestCall(rc, nil)
	must(t, err)
	assert.NotEmpty(t, r.Profiles)

	v, err := uuid.NewV4()
	must(t, err)

	_, err = messages.ProfileDataPut.TestCall(rc, &butlerd.ProfileDataPutParams{
		ProfileID: prof.ID,
		Key:       "@integrate/hello",
		Value:     v.String(),
	})
	must(t, err)

	dgr, err := messages.ProfileDataGet.TestCall(rc, &butlerd.ProfileDataGetParams{
		ProfileID: prof.ID,
		Key:       "@integrate/hello",
	})
	must(t, err)
	assert.True(t, dgr.OK)
	assert.EqualValues(t, v.String(), dgr.Value)

	dgr, err = messages.ProfileDataGet.TestCall(rc, &butlerd.ProfileDataGetParams{
		ProfileID: prof.ID,
		Key:       "@integrate/whoops",
	})
	must(t, err)
	assert.False(t, dgr.OK)
}

func authenticate(t *testing.T, rc *butlerd.RequestContext) *butlerd.Profile {
	prof, err := messages.ProfileLoginWithAPIKey.TestCall(rc, &butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: os.Getenv("ITCH_TEST_ACCOUNT_API_KEY"),
	})
	must(t, err)
	assert.EqualValues(t, "itch test account", prof.Profile.User.DisplayName)

	return prof.Profile
}
