package integrate

import (
	"os"
	"strings"
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
)

func Test_InstallLove(t *testing.T) {
	rc, h, cancel := connect(t)
	defer cancel()

	authenticate(t, rc)
	setupTmpInstallLocation(t, h, rc)

	{
		// itch-test-account/dot-love
		game := getGame(t, h, rc, 283345)

		queueRes, err := messages.InstallQueue.TestCall(rc, butlerd.InstallQueueParams{
			Game:              game,
			InstallLocationID: "tmp",
		})
		must(t, err)

		t.Logf("Queued %s", queueRes.InstallFolder)

		_, err = messages.InstallPerform.TestCall(rc, butlerd.InstallPerformParams{
			ID:            queueRes.ID,
			StagingFolder: queueRes.StagingFolder,
		})
		must(t, err)

		err = func() error {
			t.Logf("Looking inside %s to make sure we haven't extacted the .love file", queueRes.InstallFolder)

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
					t.Logf("Found it! %s", name)
					foundLove = true
					break
				}
			}

			if !foundLove {
				return errors.Errorf("Should have .love file in install folder")
			}
			return nil
		}()
		must(t, err)
	}
}
