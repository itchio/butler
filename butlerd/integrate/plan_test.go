package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Plan(t *testing.T) {
	assert := assert.New(t)

	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	_developer := store.MakeUser("Jenny Block")
	_game := _developer.MakeGame("Z-Moon")
	_game.Publish()
	_taggedUpload := _game.MakeUpload("tagged.zip")
	_taggedUpload.SetAllPlatforms()
	_taggedUpload.SetZipContents()
	_untaggedUpload := _game.MakeUpload("untagged.zip")
	_untaggedUpload.SetZipContents()

	res, err := messages.InstallPlan.TestCall(rc, butlerd.InstallPlanParams{
		DownloadSessionID: "test",
		GameID:            _game.ID,
		UploadID:          _untaggedUpload.ID,
	})
	assert.NoError(err)

	assert.NotNil(res.Info)
	assert.NotNil(res.Info.Upload)
	assert.NotEqual(_untaggedUpload.ID, res.Info.Upload.ID)
}
