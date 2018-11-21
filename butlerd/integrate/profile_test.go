package integrate

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/itchio/butler/butlerd"

	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Profile(t *testing.T) {
	rc, _, cancel := newInstance(t).Unwrap()
	defer cancel()

	_, err := messages.ProfileLoginWithAPIKey.TestCall(rc, butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: "meh",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "itch.io API error (403)")

	prof := authenticate(t, rc)

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

func authenticate(t *testing.T, rc *butlerd.RequestContext) *butlerd.Profile {
	envVar := "ITCH_TEST_ACCOUNT_API_KEY"
	apiKey := os.Getenv(envVar)
	if apiKey == "" {
		panic(fmt.Sprintf("$%s must be set to run integration tests", envVar))
	}

	prof, err := messages.ProfileLoginWithAPIKey.TestCall(rc, butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: apiKey,
	})
	must(t, err)
	assert.EqualValues(t, "itch test account", prof.Profile.User.DisplayName)

	return prof.Profile
}
