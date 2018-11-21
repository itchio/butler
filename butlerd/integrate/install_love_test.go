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
	bi := newInstance(t)
	rc, h, cancel := bi.Unwrap()
	defer cancel()

	authenticate(t, rc)

	{
		// itch-test-account/dot-love
		game := getGame(t, h, rc, 283345)

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

		err = func() error {
			bi.Logf("Looking inside %s to make sure we haven't extacted the .love file", queueRes.InstallFolder)

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
					bi.Logf("Found it! %s", name)
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
