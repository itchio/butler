package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Test_InstallSmall(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)
	setupTmpInstallLocation(t, h, rc)

	{
		// itch-test-account/111-first
		game := getGame(t, h, rc, 149766)

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

		_, err = messages.Launch.TestCall(rc, &butlerd.LaunchParams{
			CaveID: queueRes.CaveID,
		})
		must(t, err)
	}
}
