package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/stretchr/testify/assert"
)

func Test_Plan(t *testing.T) {
	assert := assert.New(t)

	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)
	setupTmpInstallLocation(t, h, rc)

	var untaggedUploadID int64 = 1111754

	res, err := messages.InstallPlan.TestCall(rc, butlerd.InstallPlanParams{
		// https://itch-test-account.itch.io/one-untagged
		GameID: 323326,

		DownloadSessionID: "test",

		// the untagged upload
		UploadID: untaggedUploadID,
	})
	assert.NoError(err)

	assert.NotNil(res.Info)
	assert.NotNil(res.Info.Upload)
	assert.NotEqual(untaggedUploadID, res.Info.Upload.ID)
}
