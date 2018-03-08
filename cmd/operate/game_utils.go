package operate

import (
	"fmt"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/manager"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	itchio "github.com/itchio/go-itchio"
)

func GameToString(game *itchio.Game) string {
	if game == nil {
		return "<nil game>"
	}

	return fmt.Sprintf("%s - %s", game.Title, game.URL)
}

func GetFilteredUploads(client *itchio.Client, game *itchio.Game, credentials *buse.GameCredentials, consumer *state.Consumer) (*manager.NarrowDownUploadsResult, error) {
	uploads, err := client.ListGameUploads(&itchio.ListGameUploadsParams{
		GameID:        game.ID,
		DownloadKeyID: credentials.DownloadKey,
	})
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	numInputs := len(uploads.Uploads)
	if numInputs == 0 {
		consumer.Infof("No uploads found at all (that we can access)")
	}
	uploadsFilterResult := manager.NarrowDownUploads(uploads.Uploads, game, manager.CurrentRuntime())

	numResults := len(uploadsFilterResult.Uploads)

	if numInputs > 0 {
		if numResults == 0 {
			consumer.Infof("→ All uploads were filtered out")
		}
		qualif := fmt.Sprintf("these %d uploads", numResults)
		if numResults == 1 {
			qualif = "this upload"
		}

		consumer.Infof("→ Narrowed %d uploads down to %s: ", numInputs, qualif)
		for _, u := range uploadsFilterResult.Uploads {
			LogUpload(consumer, u, u.Build)
		}
	}

	return uploadsFilterResult, nil
}

func LogUpload(consumer *state.Consumer, u *itchio.Upload, b *itchio.Build) {
	if u == nil {
		consumer.Infof("  No upload")
	} else {
		var name string
		if u.DisplayName != "" {
			name = u.DisplayName
		} else {
			name = u.Filename
		}

		var size string
		if u.Size > 0 {
			size = humanize.IBytes(uint64(u.Size))
		} else {
			size = "Unknown size"
		}

		consumer.Infof("  ☁ %s :: %s :: #%d", name, size, u.ID)

		var plats []string
		if u.Linux {
			plats = append(plats, "Linux")
		}
		if u.Windows {
			plats = append(plats, "Windows")
		}
		if u.OSX {
			plats = append(plats, "macOS")
		}
		if u.Android {
			plats = append(plats, "Android")
		}

		var platString = "No platforms"
		if len(plats) > 0 {
			platString = strings.Join(plats, ", ")
		}

		consumer.Infof("    %s :: %s", formatUploadType(u.Type), platString)
	}

	if b != nil {
		version := ""
		if b.UserVersion != "" {
			version = b.UserVersion
		} else if b.Version != 0 {
			version = "No explicit version"
		}

		consumer.Infof("    Build %d for channel (%s) :: %s :: #%d", b.Version, u.ChannelName, version, b.ID)
	}
}

func formatUploadType(uploadType string) string {
	switch uploadType {
	case "default":
		return "Executable"

	case "flash":
		return "Flash object"
	case "unity":
		return "Legacy Unity Web"
	case "java":
		return "Java applet"

	case "soundtrack":
		return "Soundtrack"
	case "book":
		return "Book"
	case "video":
		return "Video"
	case "documentation":
		return "Documentation"
	case "mod":
		return "Mod"
	case "audio_assets":
		return "Audio assets"
	case "graphical_assets":
		return "Graphical assets"
	case "sourcecode":
		return "Source code"

	case "other":
		return "Other"

	default:
		return fmt.Sprintf("(%s)", uploadType)
	}
}

func CredentialsForGame(db *gorm.DB, consumer *state.Consumer, game *itchio.Game) (*buse.GameCredentials, error) {
	// look for owner access
	{
		pgs, err := models.ProfileGamesByGameID(db, game.ID)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		if len(pgs) > 0 {
			pg := pgs[0]
			consumer.Infof("%s is owned by user #%d, so they must have full access", GameToString, pg.UserID)
			p, err := models.ProfileByID(db, pg.UserID)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			creds := &buse.GameCredentials{
				APIKey: p.APIKey,
			}
			return creds, nil
		}
	}

	// look for press access
	if game.InPressSystem {
		profiles, err := models.AllProfiles(db)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		for _, p := range profiles {
			if p.PressUser {
				consumer.Infof("%s is in press system and user #%d is a press user", GameToString(game), p.UserID)
				creds := &buse.GameCredentials{
					APIKey: p.APIKey,
				}
				return creds, nil
			}
		}
	}

	// look for a download key
	{
		dks, err := models.DownloadKeysByGameID(db, game.ID)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if len(dks) > 0 {
			dk := dks[0]
			p, err := models.ProfileByID(db, dk.OwnerID)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}

			consumer.Infof("%s has a download key belonging to user #%d", GameToString(game), p.UserID)
			creds := &buse.GameCredentials{
				APIKey:      p.APIKey,
				DownloadKey: dk.ID,
			}
			return creds, nil
		}
	}

	// no special credentials
	{
		consumer.Infof("%s is not related to any known profiles", GameToString(game))
		profiles, err := models.AllProfiles(db)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		if len(profiles) == 0 {
			return nil, errors.New("No profiles found")
		}

		p := profiles[0]
		creds := &buse.GameCredentials{
			APIKey: p.APIKey,
		}
		return creds, nil
	}
}

func ValidateCave(rc *buse.RequestContext, caveID string) (*models.Cave, *gorm.DB, error) {
	if caveID == "" {
		return nil, nil, errors.New("caveId must be set")
	}

	db, err := rc.DB()
	if err != nil {
		return nil, nil, errors.Wrap(err, 0)
	}

	cave, err := models.CaveByID(db, caveID)
	if err != nil {
		return nil, nil, errors.Wrap(err, 0)
	}

	if cave == nil {
		return nil, nil, fmt.Errorf("cave not found: (%s)", caveID)
	}

	return cave, db, nil
}
