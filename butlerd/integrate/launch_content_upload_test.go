package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/dash"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A cart uploaded as source code: the folder is offered first, so
// nothing runs unasked, and the cart is still there for a client with a
// runtime for it.
func Test_LaunchContentUpload(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()
	profile := bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Paranoid Cactus")
	_game := _developer.MakeGame("UFO Swamp Odyssey")
	_game.Publish()
	_upload := _game.MakeUpload("cart")
	_upload.Type = "sourcecode"
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry("ufo.p8").String("pico-8 cartridge // http://www.pico-8.com\nversion 27\n__lua__\nfunction _draw() cls() end\n")
	})

	var launched []butlerd.RuntimeLaunchParams
	messages.RuntimeLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.RuntimeLaunchParams) (*butlerd.RuntimeLaunchResult, error) {
		launched = append(launched, params)
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

	// without a runtime the cart is not a target, so the folder is all
	// there is and it opens as before
	plainRes, err := messages.LaunchGetTargets.TestCall(rc, butlerd.LaunchGetTargetsParams{
		CaveID: queueRes.CaveID,
	})
	must(err)
	require.Len(t, plainRes.Targets, 1)
	assert.EqualValues(butlerd.LaunchStrategyShell, plainRes.Targets[0].Strategy.Strategy)

	targetsRes, err := messages.LaunchGetTargets.TestCall(rc, butlerd.LaunchGetTargetsParams{
		CaveID:   queueRes.CaveID,
		Runtimes: []string{"pico8-cart"},
	})
	must(err)
	require.Len(t, targetsRes.Targets, 2)
	assert.EqualValues("Open folder", targetsRes.Targets[0].Action.Name)
	assert.EqualValues("folder-open", targetsRes.Targets[0].Action.Icon)
	assert.EqualValues(butlerd.LaunchStrategyShell, targetsRes.Targets[0].Strategy.Strategy)
	cart := targetsRes.Targets[1]
	assert.EqualValues(butlerd.LaunchStrategyRuntime, cart.Strategy.Strategy)
	assert.EqualValues(dash.FlavorPico8Cart, cart.Strategy.Candidate.Flavor)

	// a client that cannot open folders is left with the cart alone
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:            queueRes.CaveID,
		ProfileID:         profile.ID,
		Runtimes:          []string{"pico8-cart"},
		AllowedStrategies: []butlerd.LaunchStrategy{butlerd.LaunchStrategyRuntime},
	})
	must(err)
	require.Len(t, launched, 1)
	assert.EqualValues(cart.Strategy.FullTargetPath, launched[0].FullTargetPath)

	// source code with a manifest for a build that was never made: the
	// manifest fails to resolve, and the folder must still open
	_source := _developer.MakeGame("Moon Cannon source")
	_source.Publish()
	_sourceUpload := _source.MakeUpload("source")
	_sourceUpload.Type = "sourcecode"
	_sourceUpload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry("main.c").String("int main(void) { return 0; }")
		ac.Entry(".itch.toml").String(`
[[actions]]
name = "play"
path = "mooncannon{{EXT}}"
`)
	})
	sourceQueue, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              bi.FetchGame(_source.ID),
		InstallLocationID: "tmp",
	})
	must(err)
	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            sourceQueue.ID,
		StagingFolder: sourceQueue.StagingFolder,
	})
	must(err)
	sourceRes, err := messages.LaunchGetTargets.TestCall(rc, butlerd.LaunchGetTargetsParams{
		CaveID:   sourceQueue.CaveID,
		Runtimes: []string{"pico8-cart"},
	})
	must(err)
	require.Len(t, sourceRes.Targets, 1)
	assert.EqualValues("Open folder", sourceRes.Targets[0].Action.Name)
	assert.EqualValues(butlerd.LaunchStrategyShell, sourceRes.Targets[0].Strategy.Strategy)
}
