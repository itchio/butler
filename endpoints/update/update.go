package update

import (
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/buse/messages"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/cmd/operate/memorylogger"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
)

func Register(router *buse.Router) {
	messages.CheckUpdate.Register(router, func(rc *buse.RequestContext, params *buse.CheckUpdateParams) (*buse.CheckUpdateResult, error) {
		consumer := rc.Consumer
		res := &buse.CheckUpdateResult{}

		for _, item := range params.Items {
			ml := memorylogger.New()
			update, err := checkUpdateItem(rc, ml.Consumer(), item)
			if err != nil {
				res.Warnings = append(res.Warnings, err.Error())
				if se, ok := err.(*errors.Error); ok {
					consumer.Warnf("An update check failed: %s", se.ErrorStack())
				} else {
					consumer.Warnf("An update check failed: %s", err.Error())
				}
				consumer.Warnf("Log follows:")
				ml.Copy(consumer)
				consumer.Warnf("End of log")
			} else {
				if update != nil {
					res.Updates = append(res.Updates, update)
					err := messages.GameUpdateAvailable.Notify(rc, &buse.GameUpdateAvailableNotification{
						Update: update,
					})
					if err != nil {
						consumer.Warnf("Could not send GameUpdateAvailable notification: %s", err.Error())
					}
				}
			}
		}

		return res, nil
	})
}

func checkUpdateItem(rc *buse.RequestContext, consumer *state.Consumer, item *buse.CheckUpdateItem) (*buse.GameUpdate, error) {
	// TODO: respect upload successors, use upcoming check-update endpoint

	if item.ItemID == "" {
		return nil, errors.New("missing itemId")
	}

	if item.Credentials == nil {
		return nil, errors.New("missing credentials")
	}

	if item.Game == nil {
		return nil, errors.New("missing game")
	}

	if item.Upload == nil {
		return nil, errors.New("missing upload")
	}

	consumer.Statf("Checking for updates to (%s)", operate.GameToString(item.Game))
	consumer.Statf("Item ID (%s)", item.ItemID)
	consumer.Infof("→ Cached upload:")
	operate.LogUpload(consumer, item.Upload, item.Build)

	if item.Credentials.DownloadKey > 0 {
		consumer.Infof("→ Has download key (game is owned)")
	} else {
		consumer.Infof("→ Searching without download key")
	}

	installedAt, err := buse.FromDateTime(item.InstalledAt)
	if err != nil {
		consumer.Warnf("Could not parse installedAt: %s", err.Error())
		installedAt = time.Unix(0, 0)
		consumer.Warnf("Assuming unix 0 time")
	}
	consumer.Infof("→ Last install operation at (%s)", installedAt)

	client, err := rc.Harness.ClientFromCredentials(item.Credentials)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	listUploadsRes, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
		GameID: item.Game.ID,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var freshUpload *itchio.Upload
	for _, u := range listUploadsRes.Uploads {
		if u.ID == item.Upload.ID {
			freshUpload = u
			break
		}
	}

	if freshUpload == nil {
		consumer.Infof("No update found (upload disappeared)")
		return nil, nil
	}

	consumer.Infof("→ Fresh upload:")
	operate.LogUpload(consumer, freshUpload, freshUpload.Build)

	// non-wharf updates
	if item.Build == nil {
		consumer.Infof("We have no build installed, comparing timestamps...")
		if freshUpload.Build != nil {
			return nil, errors.New("We have no build installed but fresh upload has one. This shouldn't happen")
		}

		// TODO: don't do that, use the upload's hashes instead

		updatedAt, err := itchio.ParseDate(freshUpload.UpdatedAt)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		consumer.Infof("→ Upload updated at (%s)", installedAt)

		if updatedAt.After(installedAt) {
			consumer.Statf("↑ Upload was updated after last install, it's an update!")
			res := &buse.GameUpdate{
				ItemID: item.ItemID,
				Game:   item.Game,
				Upload: freshUpload,
				Build:  nil,
			}
			return res, nil
		}
		return nil, nil
	}

	// wharf updates
	{
		consumer.Infof("We have no build installed, comparing build numbers...")
		if freshUpload.Build == nil {
			return nil, errors.New("We have a build installed but fresh upload has none. This shouldn't happen")
		}

		if freshUpload.Build.ID > item.Build.ID {
			consumer.Statf("↑ A more recent build (#%d) than ours (#%d) is available, it's an update!",
				freshUpload.Build.ID,
				item.Build.ID,
			)
			res := &buse.GameUpdate{
				ItemID: item.ItemID,
				Game:   item.Game,
				Upload: freshUpload,
				Build:  freshUpload.Build,
			}
			return res, nil
		}
	}

	return nil, nil
}
