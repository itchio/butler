package integrate

import (
	"os"
	"strings"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
)

func Test_InstallLove(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Kernel Panic")
	_game := _developer.MakeGame("dot love")
	_game.Publish()
	_upload := _game.MakeUpload("All platforms")
	_upload.SetAllPlatforms()
	_upload.SetZipContentsCustom(func(ac *mitch.ArchiveContext) {
		ac.SetName("hello.love")
		ac.Entry("main.lua").String("print 'hello lua'")
	})

	game := bi.FetchGame(_game.ID)

	queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
		Game:              game,
		InstallLocationID: "tmp",
	})
	must(err)

	bi.Logf("Queued %s", queueRes.InstallFolder)

	_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(err)

	err = func() error {
		bi.Logf("Looking inside %s to make sure we haven't extracted the .love file", queueRes.InstallFolder)

		dir, err := os.Open(queueRes.InstallFolder)
		if err != nil {
			return err
		}
		defer dir.Close()

		names, err := dir.Readdirnames(-1)
		if err != nil {
			return err
		}

		foundLove := false
		for _, name := range names {
			if strings.HasSuffix(strings.ToLower(name), ".love") {
				bi.Logf("Found it! %s", name)
				foundLove = true
				break
			}
		}

		assert.True(foundLove, "should have found .love file in install folder")
		return nil
	}()
	must(err)
}
