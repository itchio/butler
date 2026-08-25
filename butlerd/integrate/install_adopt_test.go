package integrate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/hush/bfs"
	"github.com/itchio/mitch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InstallAdopt(t *testing.T) {
	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	developer := store.MakeUser("Adoptive Parent")
	gameRecord := developer.MakeGame("Already Here")
	gameRecord.Type = "html"
	gameRecord.Publish()
	uploadRecord := gameRecord.MakeUpload("External untagged upload")
	uploadRecord.Storage = "external"
	uploadRecord.URL = "https://example.com/already-here.zip"
	uploadRecord.Filename = "already-here.zip"

	wd, err := os.Getwd()
	require.NoError(t, err)
	installFolder := filepath.Join(wd, "tmp", "already-here")
	require.NoError(t, os.MkdirAll(installFolder, 0o755))
	indexPath := filepath.Join(installFolder, "index.html")
	require.NoError(t, os.WriteFile(indexPath, []byte("<p>local copy</p>"), 0o644))

	res, err := messages.InstallAdopt.TestCall(rc, butlerd.InstallAdoptParams{
		GameID:            gameRecord.ID,
		UploadID:          uploadRecord.ID,
		InstallLocationID: "tmp",
		InstallFolderName: "already-here",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Cave)
	assert.Equal(t, gameRecord.ID, res.Cave.Game.ID)
	assert.Equal(t, uploadRecord.ID, res.Cave.Upload.ID)
	assert.Equal(t, installFolder, res.Cave.InstallInfo.InstallFolder)
	assert.Positive(t, res.Cave.InstallInfo.InstalledSize)
	assert.NotNil(t, res.Cave.Stats.InstalledAt)

	receipt, err := bfs.ReadReceipt(installFolder)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.Equal(t, gameRecord.ID, receipt.Game.ID)
	assert.Equal(t, uploadRecord.ID, receipt.Upload.ID)
	assert.NotNil(t, receipt.Files)
	assert.Empty(t, receipt.Files)

	fetched, err := messages.FetchCave.TestCall(rc, butlerd.FetchCaveParams{
		CaveID: res.Cave.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, fetched.Cave)
	assert.Equal(t, installFolder, fetched.Cave.InstallInfo.InstallFolder)

	contents, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	assert.Equal(t, "<p>local copy</p>", string(contents))

	_, err = messages.InstallAdopt.TestCall(rc, butlerd.InstallAdoptParams{
		GameID:            gameRecord.ID,
		UploadID:          uploadRecord.ID,
		InstallLocationID: "tmp",
		InstallFolderName: "already-here",
	})
	assert.Error(t, err)
}

func Test_InstallAdoptHistoricalBuild(t *testing.T) {
	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	developer := store.MakeUser("Build Historian")
	gameRecord := developer.MakeGame("Older Local Build")
	gameRecord.Type = "html"
	gameRecord.Publish()
	uploadRecord := gameRecord.MakeUpload("Wharf channel")
	uploadRecord.SetAllPlatforms()
	oldBuild := uploadRecord.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.Entry("index.html").String("<p>old build</p>")
	})
	latestBuild := uploadRecord.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.Entry("index.html").String("<p>latest build</p>")
	})
	require.NotEqual(t, oldBuild.ID, latestBuild.ID)

	wd, err := os.Getwd()
	require.NoError(t, err)
	installFolder := filepath.Join(wd, "tmp", "older-local-build")
	require.NoError(t, os.MkdirAll(installFolder, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(installFolder, "index.html"), []byte("<p>old build</p>"), 0o644))

	res, err := messages.InstallAdopt.TestCall(rc, butlerd.InstallAdoptParams{
		GameID:            gameRecord.ID,
		UploadID:          uploadRecord.ID,
		BuildID:           oldBuild.ID,
		InstallLocationID: "tmp",
		InstallFolderName: "older-local-build",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Cave)
	require.NotNil(t, res.Cave.Build)
	assert.Equal(t, oldBuild.ID, res.Cave.Build.ID)

	receipt, err := bfs.ReadReceipt(installFolder)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.NotNil(t, receipt.Build)
	assert.Equal(t, oldBuild.ID, receipt.Build.ID)
}

func Test_InstallAdoptRejectsUnsafeOrMismatchedFolders(t *testing.T) {
	bi := newInstance(t)
	rc, _, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	store := bi.Server.Store()
	developer := store.MakeUser("Careful Curator")
	gameRecord := developer.MakeGame("Expected Game")
	gameRecord.Publish()
	uploadRecord := gameRecord.MakeUpload("Expected Upload")
	uploadRecord.SetAllPlatforms()
	otherGame := developer.MakeGame("Other Game")
	otherGame.Publish()
	otherUpload := otherGame.MakeUpload("Other Upload")
	otherUpload.SetAllPlatforms()
	otherBuild := otherUpload.PushBuild(func(ac *mitch.ArchiveContext) {
		ac.Entry("other.txt").String("belongs to another upload")
	})

	wd, err := os.Getwd()
	require.NoError(t, err)
	tmpLocation := filepath.Join(wd, "tmp")
	require.NoError(t, os.MkdirAll(filepath.Join(tmpLocation, "safe-folder"), 0o755))

	baseParams := butlerd.InstallAdoptParams{
		GameID:            gameRecord.ID,
		UploadID:          uploadRecord.ID,
		InstallLocationID: "tmp",
	}

	for _, folderName := range []string{".", "..", "../outside", "nested/folder", `nested\\folder`, "downloads"} {
		params := baseParams
		params.InstallFolderName = folderName
		_, err := messages.InstallAdopt.TestCall(rc, params)
		assert.Error(t, err, folderName)
	}

	params := baseParams
	params.InstallFolderName = "missing"
	_, err = messages.InstallAdopt.TestCall(rc, params)
	assert.Error(t, err)

	params = baseParams
	params.InstallFolderName = "safe-folder"
	params.BuildID = otherBuild.ID
	_, err = messages.InstallAdopt.TestCall(rc, params)
	assert.Error(t, err)
	_, err = os.Stat(bfs.ReceiptPath(filepath.Join(tmpLocation, "safe-folder")))
	assert.True(t, os.IsNotExist(err))

	params.InstallFolderName = "safe-folder"
	params.UploadID = otherUpload.ID
	_, err = messages.InstallAdopt.TestCall(rc, params)
	assert.Error(t, err)

	if runtime.GOOS != "windows" {
		require.NoError(t, os.MkdirAll(filepath.Join(tmpLocation, "symlink-target"), 0o755))
		require.NoError(t, os.Symlink(filepath.Join(tmpLocation, "symlink-target"), filepath.Join(tmpLocation, "symlink-folder")))
		params = baseParams
		params.InstallFolderName = "symlink-folder"
		_, err = messages.InstallAdopt.TestCall(rc, params)
		assert.Error(t, err)
	}
}
