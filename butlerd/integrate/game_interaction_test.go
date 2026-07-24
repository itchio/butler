package integrate

import (
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_GameInteraction(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()
	profile := bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Ivy Session")
	_game := _developer.MakeGame("Timesink")
	_game.Type = "html"
	_game.Publish()
	_upload := _game.MakeUpload("web build")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry(".itch.toml").String(`
[[actions]]
name = "play"
path = "index.html"
`)
		ac.Entry("index.html").String("<p>Hi!</p>")
	})

	messages.HTMLLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.HTMLLaunchParams) (*butlerd.HTMLLaunchResult, error) {
		return &butlerd.HTMLLaunchResult{}, nil
	})

	game := bi.FetchGame(_game.ID)
	queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		InstallLocationID: "tmp",
	})
	must(err)
	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	const bogusProfileID = 999999

	assertNoSuchProfile := func(err error) {
		assert.Error(err)
		je := err.(*jsonrpc2.Error)
		assert.EqualValues(butlerd.CodeNoSuchProfile, je.Code)
	}

	_, err = messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: bogusProfileID,
		GameID:    _game.ID,
	})
	assertNoSuchProfile(err)

	_, err = messages.FetchCaves.TestCall(rc, butlerd.FetchCavesParams{
		ProfileID: bogusProfileID,
	})
	assertNoSuchProfile(err)

	_, err = messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
		CaveID:    queueRes.CaveID,
		ProfileID: bogusProfileID,
	})
	assertNoSuchProfile(err)

	_, err = messages.FetchCommons.TestCall(rc, butlerd.FetchCommonsParams{
		ProfileID: bogusProfileID,
	})
	assertNoSuchProfile(err)

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
		ProfileID:  bogusProfileID,
	})
	assertNoSuchProfile(err)

	interactionRes, err := messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: profile.ID,
		GameID:    _game.ID,
	})
	must(err)
	assert.Nil(interactionRes.Interaction)
	assert.True(interactionRes.Stale)

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
		ProfileID:  profile.ID,
	})
	must(err)

	interactionRes, err = messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: profile.ID,
		GameID:    _game.ID,
	})
	must(err)
	assert.NotNil(interactionRes.Interaction)
	assert.False(interactionRes.Stale)
	assert.EqualValues(profile.ID, interactionRes.Interaction.UserID)
	assert.EqualValues(_game.ID, interactionRes.Interaction.GameID)
	assert.NotNil(interactionRes.Interaction.SyncedAt)

	caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
	assert.NotNil(caveRes.Cave.Stats.LocalLastRunAt)

	cavesRes, err := messages.FetchCaves.TestCall(rc, butlerd.FetchCavesParams{
		ProfileID: profile.ID,
	})
	must(err)
	assert.Len(cavesRes.Items, 1)
	assert.NotNil(cavesRes.Items[0].Interaction)
	assert.EqualValues(profile.ID, cavesRes.Items[0].Interaction.UserID)

	cavesRes, err = messages.FetchCaves.TestCall(rc, butlerd.FetchCavesParams{})
	must(err)
	assert.Len(cavesRes.Items, 1)
	assert.Nil(cavesRes.Items[0].Interaction)

	commonsRes, err := messages.FetchCommons.TestCall(rc, butlerd.FetchCommonsParams{
		ProfileID: profile.ID,
	})
	must(err)
	assert.Len(commonsRes.Caves, 1)
	assert.NotNil(commonsRes.Caves[0].Interaction)
	assert.NotNil(commonsRes.Caves[0].LocalLastRunAt)
}

func Test_GameInteractionMultiProfile(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()
	profileA := bi.Authenticate()

	store := bi.Server.Store()

	_userB := store.MakeUser("Second Account")
	_apiKeyB := _userB.MakeAPIKey()
	profB, err := messages.ProfileLoginWithAPIKey.TestCall(rc, butlerd.ProfileLoginWithAPIKeyParams{
		APIKey: _apiKeyB.Key,
	})
	must(err)
	profileB := profB.Profile

	_developer := store.MakeUser("Ivy Session")
	_game := _developer.MakeGame("Timesink")
	_game.Publish()

	_userA := store.FindUser(profileA.ID)
	_userA.MakeGameSession(_game.ID, 5000, time.Now())
	_userA.MakeGameSession(_game.ID, 300, time.Now())
	_userB.MakeGameSession(_game.ID, 111, time.Now())

	resA, err := messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: profileA.ID,
		GameID:    _game.ID,
		Fresh:     true,
	})
	must(err)
	assert.NotNil(resA.Interaction)
	assert.EqualValues(5300, resA.Interaction.SecondsRun)
	assert.EqualValues(profileA.ID, resA.Interaction.UserID)

	resB, err := messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: profileB.ID,
		GameID:    _game.ID,
		Fresh:     true,
	})
	must(err)
	assert.NotNil(resB.Interaction)
	assert.EqualValues(111, resB.Interaction.SecondsRun)
	assert.EqualValues(profileB.ID, resB.Interaction.UserID)

	resA, err = messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: profileA.ID,
		GameID:    _game.ID,
	})
	must(err)
	assert.EqualValues(5300, resA.Interaction.SecondsRun)
}
