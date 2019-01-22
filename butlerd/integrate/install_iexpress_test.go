package integrate

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_InstallIExpress(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("IE6 Apparentlyj")
	_game := _developer.MakeGame("natas")
	_game.Publish()
	_upload := _game.MakeUpload("Ahhhh")
	_upload.SetAllPlatforms()

	iexpressBytes, err := ioutil.ReadFile("testdata/iexpress-sample.exe")
	must(err)
	_upload.SetHostedContents("iexpress-sample.exe", iexpressBytes)

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
		bi.Logf("Looking inside %s to make sure we have indeed extracted hello.txt", queueRes.InstallFolder)

		dir, err := os.Open(queueRes.InstallFolder)
		if err != nil {
			return err
		}
		defer dir.Close()

		names, err := dir.Readdirnames(-1)
		if err != nil {
			return err
		}

		foundTxt := false
		for _, name := range names {
			if strings.HasSuffix(strings.ToLower(name), ".txt") {
				bi.Logf("Found it! %s", name)
				foundTxt = true
				break
			}
		}

		assert.True(foundTxt, "should have found .txt file in install folder")
		return nil
	}()
	must(err)
}
