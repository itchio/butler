package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"

	"github.com/itchio/mitch"
)

func Test_ChangeUploadType(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()
	bi.Authenticate()

	s := bi.Server.Store()
	_developer := s.MakeUser("Aaaa")
	_game := _developer.MakeGame("Gens Cachés")
	_game.Publish()
	_upload := _game.MakeUpload("web version")
	_upload.SetAllPlatforms()
	_upload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.Entry("song1.ogg").Random(0xfeed0001, 512*1024)
		ac.Entry("song2.ogg").Random(0xfeed0002, 512*1024)
		ac.Entry("song3.ogg").Random(0xfeed0003, 512*1024)
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

	bi.Logf("Launching with wrong upload type")
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "./tmp/prereqs",
	})
	assert.Error(err)
	bi.Logf("Got error, as expected: %v", err)
	assert.Contains(err.Error(), "Nothing that can be launched was found")

	bi.Logf("Changing upload type...")
	_upload.Type = "soundtrack"

	bi.Logf("Registering shell launch handler...")
	hadShellLaunch := false
	messages.ShellLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.ShellLaunchParams) (*butlerd.ShellLaunchResult, error) {
		hadShellLaunch = true
		bi.Logf("Performing shell launch!")
		return &butlerd.ShellLaunchResult{}, nil
	})

	bi.Logf("Launching again...")
	_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
		CaveID:     queueRes.CaveID,
		PrereqsDir: "./tmp/prereqs",
	})
	assert.NoError(err, "launch went fine")
	assert.True(hadShellLaunch, "had shell launch")
}
