package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/require"
)

func Test_Plan(t *testing.T) {
	require := require.New(t)

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

	// Test Install.Plan (deprecated): verify Info is populated
	planRes, err := messages.InstallPlan.TestCall(rc, butlerd.InstallPlanParams{
		GameID: _game.ID,
	})
	require.NoError(err)
	require.NotNil(planRes.Game)
	require.NotEmpty(planRes.Uploads)
	require.NotNil(planRes.Info)
	require.NotNil(planRes.Info.Upload)
	require.NotNil(planRes.Info.DiskUsage)

	// Test Install.GetUploads: fast path, no Info
	getUploadsRes, err := messages.InstallGetUploads.TestCall(rc, butlerd.InstallGetUploadsParams{
		GameID: _game.ID,
	})
	require.NoError(err)
	require.NotNil(getUploadsRes.Game)
	require.NotEmpty(getUploadsRes.Uploads)

	// Test Install.PlanUpload: get info for a specific upload
	upload := getUploadsRes.Uploads[0]

	infoRes, err := messages.InstallPlanUpload.TestCall(rc, butlerd.InstallPlanUploadParams{
		UploadID:          upload.ID,
		DownloadSessionID: "test",
	})
	require.NoError(err)
	require.NotNil(infoRes.Info)
	require.NotNil(infoRes.Info.Upload)
	require.NotNil(infoRes.Info.DiskUsage)
}
