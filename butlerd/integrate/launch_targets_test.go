package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/jsonrpc2"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_LaunchTargets(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()
	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Peggy Sample")
	_game := _developer.MakeGame("Target Practice")
	_game.Type = "html"
	_game.Publish()
	_upload := _game.MakeUpload("All platforms")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.Entry(".itch.toml").String(`
[[actions]]
name = "play"
path = "index.html"

[[actions]]
name = "help"
path = "help.html"
`)
		ac.Entry("index.html").String("<p>Hi!</p>")
		ac.Entry("help.html").String("<p>Help!</p>")
	})

	var launched []string
	messages.HTMLLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.HTMLLaunchParams) (*butlerd.HTMLLaunchResult, error) {
		launched = append(launched, params.IndexPath)
		return &butlerd.HTMLLaunchResult{}, nil
	})

	picks := 0
	messages.PickManifestAction.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.PickManifestActionParams) (*butlerd.PickManifestActionResult, error) {
		picks++
		return &butlerd.PickManifestActionResult{Index: 0}, nil
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
		CaveID: queueRes.CaveID,
	})
	must(err)
	assert.EqualValues(2, len(targetsRes.Targets))
	assert.EqualValues("play", targetsRes.Targets[0].Action.Name)
	assert.EqualValues("help", targetsRes.Targets[1].Action.Name)

	// an explicit target launches without asking the client to pick
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
		Target:     "help",
	})
	must(err)
	assert.EqualValues([]string{"help.html"}, launched)
	assert.EqualValues(0, picks)

	// an explicit target that matches nothing errors
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
		Target:     "no-such-target",
	})
	assert.Error(err)
	je := err.(*jsonrpc2.Error)
	assert.EqualValues(butlerd.CodeLaunchTargetNotFound, je.Code)
	assert.EqualValues(0, picks)

	// a valid persisted preference skips the picker
	_, err = messages.CavesSetSettings.TestCall(rc, butlerd.CavesSetSettingsParams{
		CaveID: queueRes.CaveID,
		Settings: &butlerd.CaveSettings{
			LaunchTarget: "help",
		},
	})
	must(err)

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
	})
	must(err)
	assert.EqualValues([]string{"help.html", "help.html"}, launched)
	assert.EqualValues(0, picks)

	// a stale persisted preference falls back to normal selection
	// (here: the picker, since there are two targets)
	_, err = messages.CavesSetSettings.TestCall(rc, butlerd.CavesSetSettingsParams{
		CaveID: queueRes.CaveID,
		Settings: &butlerd.CaveSettings{
			LaunchTarget: "went-stale-after-update",
		},
	})
	must(err)

	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "/tmp/prereqs",
	})
	must(err)
	assert.EqualValues(1, picks)
	assert.EqualValues([]string{"help.html", "help.html", "index.html"}, launched)
}
