package integrate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
)

func Test_Install(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)

	var game *itchio.Game
	messages.FetchGameYield.Register(h, func(rc *butlerd.RequestContext, params *butlerd.FetchGameYieldNotification) {
		game = params.Game
	})

	_, err := messages.FetchGame.TestCall(rc, &butlerd.FetchGameParams{
		GameID: 149766,
	})
	must(t, err)
	assert.True(t, game != nil)

	must(t, os.RemoveAll("./tmp"))

	_, err = messages.InstallLocationsAdd.TestCall(rc, &butlerd.InstallLocationsAddParams{
		ID:   "tmp",
		Path: "./tmp",
	})
	must(t, err)

	queueRes, err := messages.InstallQueue.TestCall(rc, &butlerd.InstallQueueParams{
		Game:              game,
		InstallLocationID: "tmp",
	})
	must(t, err)

	t.Logf("Queued %s", queueRes.InstallFolder)

	_, err = messages.InstallPerform.TestCall(rc, &butlerd.InstallPerformParams{
		ID:            queueRes.ID,
		StagingFolder: queueRes.StagingFolder,
	})
	must(t, err)
}
