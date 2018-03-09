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

func CredentialsForGame(db *gorm.DB, consumer *state.Consumer, game *itchio.Game) *buse.GameCredentials {
	// look for owner access
	{
		pgs, err := models.ProfileGamesByGameID(db, game.ID)
		if err != nil {
			panic(err)
		}
		if len(pgs) > 0 {
			pg := pgs[0]
			consumer.Infof("%s is owned by user #%d, so they must have full access", GameToString(game), pg.ProfileID)
			p := models.ProfileByID(db, pg.ProfileID)

			if p == nil {
				consumer.Infof("Ah, we dont have a profile for #%d, nevermind", pg.ProfileID)
			} else {
				creds := &buse.GameCredentials{
					APIKey: p.APIKey,
				}
				return creds
			}
		}
	}

	// look for press access
	if game.InPressSystem {
		for _, profile := range models.AllProfiles(db) {
			if profile.PressUser {
				consumer.Infof("%s is in press system and user #%d is a press user", GameToString(game), profile.UserID)
				creds := &buse.GameCredentials{
					APIKey: profile.APIKey,
				}
				return creds
			}
		}
	}

	// look for a download key
	{
		dks := models.DownloadKeysByGameID(db, game.ID)

		for _, dk := range dks {
			profile := models.ProfileByID(db, dk.OwnerID)
			if profile == nil {
				continue
			}

			consumer.Infof("%s has a download key belonging to user #%d", GameToString(game), profile.UserID)
			creds := &buse.GameCredentials{
				APIKey:      profile.APIKey,
				DownloadKey: dk.ID,
			}
			return creds
		}
	}

	// no special credentials
	{
		consumer.Infof("%s is not related to any known profiles", GameToString(game))
		profiles := models.AllProfiles(db)
		if len(profiles) == 0 {
			panic(errors.New("No profiles found"))
		}

		profile := profiles[0]
		creds := &buse.GameCredentials{
			APIKey: profile.APIKey,
		}
		return creds
	}
}

func ValidateCave(rc *buse.RequestContext, caveID string) *models.Cave {
	if caveID == "" {
		panic(errors.New("caveId must be set"))
	}

	cave := models.CaveByID(rc.DB(), caveID)
	if cave == nil {
		panic(fmt.Errorf("cave not found: (%s)", caveID))
	}

	cave.Preload(rc.DB())

	return cave
}
