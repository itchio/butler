package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/cmd/operate"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
)

// Preload Game, Upload and Build for a given cave
func PreloadCaves(db *gorm.DB, consumer *state.Consumer, caveOrCaves interface{}) error {
	err := hades.NewContext(db, consumer).Preload(db, &hades.PreloadParams{
		Record: caveOrCaves,
		Fields: []hades.PreloadField{
			hades.PreloadField{Name: "Game"},
			hades.PreloadField{Name: "Upload"},
			hades.PreloadField{Name: "Build"},
		},
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
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
			consumer.Infof("%s is owned by user #%d, so they must have full access", operate.GameToString, pg.UserID)
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
				consumer.Infof("%s is in press system and user #%d is a press user", operate.GameToString(game), p.UserID)
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

			consumer.Infof("%s has a download key belonging to user #%d", operate.GameToString(game), p.UserID)
			creds := &buse.GameCredentials{
				APIKey:      p.APIKey,
				DownloadKey: dk.ID,
			}
			return creds, nil
		}
	}

	// no special credentials
	{
		consumer.Infof("%s is not related to any known profiles", operate.GameToString(game))
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
