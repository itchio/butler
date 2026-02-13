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

	planRes, err := messages.InstallPlan.TestCall(rc, butlerd.InstallPlanParams{
		GameID: _game.ID,
	})
	assert.NoError(err)
	assert.NotNil(planRes.Game)
	assert.NotEmpty(planRes.Uploads)

	// Use the first upload from the plan to get disk usage info
	upload := planRes.Uploads[0]

	infoRes, err := messages.InstallPlanUpload.TestCall(rc, butlerd.InstallPlanUploadParams{
		UploadID:          upload.ID,
		DownloadSessionID: "test",
	})
	assert.NoError(err)
	assert.NotNil(infoRes.Info)
	assert.NotNil(infoRes.Info.Upload)
}
