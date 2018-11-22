package integrate

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
)

func Test_InstallSmall(t *testing.T) {
	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	bi.Authenticate()

	messages.HTMLLaunch.TestRegister(h, func(rc *butlerd.RequestContext, params butlerd.HTMLLaunchParams) (*butlerd.HTMLLaunchResult, error) {
		return &butlerd.HTMLLaunchResult{}, nil
	})

	{
		// itch-test-account/111-first
		game := getGame(t, h, rc, 149766)

		queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
		})
		must(t, err)

		bi.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		must(t, err)

		_, err = messages.Launch.TestCall(rc, butlerd.LaunchParams{
			CaveID:     queueRes.CaveID,
			PrereqsDir: "/tmp/prereqs",
		})
		must(t, err)

		_, err = messages.UninstallPerform.TestCall(rc, butlerd.UninstallPerformParams{
			CaveID: queueRes.CaveID,
		})
		must(t, err)
	}
}
