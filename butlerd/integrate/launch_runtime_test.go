package integrate

import (
	"errors"
	"testing"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/dash"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A LÖVE game the client runs itself: the launch goes through butler
// for its bookkeeping, and the client answers RuntimeLaunch when the
// game exits.
func Test_LaunchRuntime(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()
	profile := bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Lua Turner")
	_game := _developer.MakeGame("Moon Cannon")
	_game.Publish()
	_upload := _game.MakeUpload("love build")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		// the action points at the folder, so it resolves to the shell
		// strategy until the client's runtimes say otherwise
		ac.Entry(".itch.toml").String(`
[[actions]]
name = "play"
path = "."
args = ["--windowed"]
`)
		ac.Entry("main.lua").String(`function love.draw() end`)
		ac.Entry("conf.lua").String(`function love.conf(t) t.version = "11.5" end`)
	})

	var launched []butlerd.RuntimeLaunchParams
	var launchErr error
	messages.RuntimeLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.RuntimeLaunchParams) (*butlerd.RuntimeLaunchResult, error) {
		launched = append(launched, params)
		// long enough for the play time to round up to a second
		time.Sleep(1100 * time.Millisecond)
		if launchErr != nil {
			return nil, launchErr
		}
		return &butlerd.RuntimeLaunchResult{}, nil
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

	targetsRes, err := messages.LaunchGetTargets.TestCall(rc, butlerd.LaunchGetTargetsParams{
		CaveID:   queueRes.CaveID,
		Runtimes: []string{"love"},
	})
	must(err)
	require.Len(t, targetsRes.Targets, 1)
	target := targetsRes.Targets[0]
	assert.EqualValues("play", target.Action.Name)
	assert.EqualValues(butlerd.LaunchStrategyRuntime, target.Strategy.Strategy)
	assert.EqualValues(dash.FlavorLove, target.Strategy.Candidate.Flavor)

	// without the runtimes list the same action is a folder to browse
	plainRes, err := messages.LaunchGetTargets.TestCall(rc, butlerd.LaunchGetTargetsParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
	require.Len(t, plainRes.Targets, 1)
	assert.EqualValues(butlerd.LaunchStrategyShell, plainRes.Targets[0].Strategy.Strategy)

	// without the runtimes list the target does not exist at launch time
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:            queueRes.CaveID,
		ProfileID:         profile.ID,
		Target:            target.Action.Path,
		AllowedStrategies: []butlerd.LaunchStrategy{butlerd.LaunchStrategyRuntime},
	})
	assert.Error(err)
	assert.Empty(launched)

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:            queueRes.CaveID,
		ProfileID:         profile.ID,
		Target:            target.Action.Path,
		Runtimes:          []string{"love"},
		AllowedStrategies: []butlerd.LaunchStrategy{butlerd.LaunchStrategyRuntime},
	})
	must(err)
	require.Len(t, launched, 1)
	assert.EqualValues(target.Strategy.FullTargetPath, launched[0].FullTargetPath)
	assert.EqualValues(dash.FlavorLove, launched[0].Candidate.Flavor)
	assert.EqualValues([]string{"--windowed"}, launched[0].Args)

	// the run was tracked as any other
	caveRes, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
	assert.NotNil(caveRes.Cave.Stats.LocalLastRunAt)
	assert.GreaterOrEqual(caveRes.Cave.Stats.LocalSecondsRun, int64(1))

	interactionRes, err := messages.FetchGameInteraction.TestCall(rc, butlerd.FetchGameInteractionParams{
		ProfileID: profile.ID,
		GameID:    _game.ID,
	})
	must(err)
	assert.NotNil(interactionRes.Interaction)

	// a failing game fails the launch
	launchErr = errors.New("the emulator died")
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:            queueRes.CaveID,
		ProfileID:         profile.ID,
		Target:            target.Action.Path,
		Runtimes:          []string{"love"},
		AllowedStrategies: []butlerd.LaunchStrategy{butlerd.LaunchStrategyRuntime},
	})
	assert.Error(err)
	assert.Len(launched, 2)
}
