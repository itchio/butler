package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_CaveSettings_GetSetAndLaunchExtraArgs(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()
	bi.Authenticate()

	store := bi.Server.Store()
	developer := store.MakeUser("Demo User")
	game := developer.MakeGame("Settings Test Game")
	game.Type = "html"
	game.Publish()
	upload := game.MakeUpload("HTML upload")
	upload.SetAllPlatforms()
	upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry(".itch.toml").String(`
[[actions]]
name = "play"
path = "index.html"
args = ["--manifest-flag", "hello"]
		`)
		ac.Entry("index.html").String("<html><body>Hello.</body></html>")
	})

	fetchedGame := bi.FetchGame(game.ID)
	queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              fetchedGame,
		InstallLocationID: "tmp",
	})
	must(err)

	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	getInitialRes, err := messages.CavesGetSettings.TestCall(rc, butlerd.CavesGetSettingsParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
	assert.Nil(getInitialRes.Settings.Sandbox)
	assert.Nil(getInitialRes.Settings.SandboxType)
	assert.Nil(getInitialRes.Settings.SandboxNoNetwork)
	assert.Nil(getInitialRes.Settings.SandboxAllowEnv)
	assert.Len(getInitialRes.Settings.ExtraArgs, 0)

	sandbox := false
	sandboxType := butlerd.SandboxTypeAuto
	noNetwork := true
	allowEnv := []string{"PATH", "HOME"}

	_, err = messages.CavesSetSettings.TestCall(rc, butlerd.CavesSetSettingsParams{
		CaveID: queueRes.CaveID,
		Settings: &butlerd.CaveSettings{
			Sandbox:          &sandbox,
			SandboxType:      &sandboxType,
			SandboxNoNetwork: &noNetwork,
			SandboxAllowEnv:  &allowEnv,
			ExtraArgs:        []string{"--saved-arg"},
		},
	})
	must(err)

	getRes, err := messages.CavesGetSettings.TestCall(rc, butlerd.CavesGetSettingsParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
	if assert.NotNil(getRes.Settings.Sandbox) {
		assert.Equal(false, *getRes.Settings.Sandbox)
	}
	if assert.NotNil(getRes.Settings.SandboxType) {
		assert.Equal(butlerd.SandboxTypeAuto, *getRes.Settings.SandboxType)
	}
	if assert.NotNil(getRes.Settings.SandboxNoNetwork) {
		assert.Equal(true, *getRes.Settings.SandboxNoNetwork)
	}
	if assert.NotNil(getRes.Settings.SandboxAllowEnv) {
		assert.Equal([]string{"PATH", "HOME"}, *getRes.Settings.SandboxAllowEnv)
	}
	assert.Equal([]string{"--saved-arg"}, getRes.Settings.ExtraArgs)

	var gotLaunchArgs []string
	messages.HTMLLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.HTMLLaunchParams) (*butlerd.HTMLLaunchResult, error) {
		gotLaunchArgs = append(gotLaunchArgs, params.Args...)
		return &butlerd.HTMLLaunchResult{}, nil
	})

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
		ExtraArgs:  []string{"--launch-extra", "world"},
	})
	must(err)

	assert.Equal([]string{"--manifest-flag", "hello", "--launch-extra", "world"}, gotLaunchArgs)
}

func Test_CaveSettings_InvalidCaveReturnsInternalError(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()
	bi.Authenticate()

	const unknownCaveID = "unknown-cave"

	_, err := messages.CavesGetSettings.TestCall(rc, butlerd.CavesGetSettingsParams{
		CaveID: unknownCaveID,
	})
	assert.Error(err)
	je := err.(*jsonrpc2.Error)
	assert.EqualValues(jsonrpc2.CodeInternalError, je.Code)

	_, err = messages.CavesSetSettings.TestCall(rc, butlerd.CavesSetSettingsParams{
		CaveID: unknownCaveID,
		Settings: &butlerd.CaveSettings{
			ExtraArgs: []string{"--x"},
		},
	})
	assert.Error(err)
	je = err.(*jsonrpc2.Error)
	assert.EqualValues(jsonrpc2.CodeInternalError, je.Code)

	_, err = messages.CavesSetPinned.TestCall(rc, butlerd.CavesSetPinnedParams{
		CaveID: unknownCaveID,
		Pinned: true,
	})
	assert.Error(err)
	je = err.(*jsonrpc2.Error)
	assert.EqualValues(jsonrpc2.CodeInternalError, je.Code)

	_, err = messages.SnoozeCave.TestCall(rc, butlerd.SnoozeCaveParams{
		CaveID: unknownCaveID,
	})
	assert.Error(err)
	je = err.(*jsonrpc2.Error)
	assert.EqualValues(jsonrpc2.CodeInternalError, je.Code)
}
