package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func FetchProfileCollections(rc *butlerd.RequestContext, params *butlerd.FetchProfileCollectionsParams) (*butlerd.FetchProfileCollectionsResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	sendDBCollections := func() error {
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustPreload(conn, profile,
				hades.AssocWithSearch("ProfileCollections", hades.Search().OrderBy("position ASC"),
					hades.Assoc("Collection"),
				),
			)
		})

		profileCollections := profile.ProfileCollections

		var collectionIDs []int64
		collectionsByIDs := make(map[int64]*itchio.Collection)
		for _, pc := range profileCollections {
			c := pc.Collection
			collectionIDs = append(collectionIDs, c.ID)
			collectionsByIDs[c.ID] = c
		}

		var rows []struct {
			itchio.CollectionGame `hades:"squash"`
			itchio.Game           `hades:"squash"`
		}
		rc.WithConn(func(conn *sqlite.Conn) {
			c := models.HadesContext()
			models.MustExecRaw(conn, `
			SELECT collection_games.*, games.*
			FROM collections
			JOIN collection_games ON collection_games.collection_id = collections.id
			JOIN games ON games.id = collection_games.game_id
			WHERE collections.id IN (?)
			AND collection_games.game_id IN (
				SELECT game_id
				FROM collection_games
				WHERE collection_games.collection_id = collections.id
				ORDER BY "position" ASC
				LIMIT 8
			)
		`, c.IntoRowsScanner(&rows), collectionIDs)
		})

		for _, row := range rows {
			c := collectionsByIDs[row.CollectionGame.CollectionID]
			cg := row.CollectionGame
			game := row.Game
			cg.Game = &game
			c.CollectionGames = append(c.CollectionGames, &cg)
		}

		if len(profileCollections) > 0 {
			yn := &butlerd.FetchProfileCollectionsYieldNotification{}
			yn.Offset = 0
			yn.Total = int64(len(profileCollections))

			for _, pc := range profileCollections {
				yn.Items = append(yn.Items, pc.Collection)
			}

			err := messages.FetchProfileCollectionsYield.Notify(rc, yn)
			if err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}

	err := sendDBCollections()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	collRes, err := client.ListProfileCollections()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	profile.ProfileCollections = nil
	for i, c := range collRes.Collections {
		for _, cg := range c.CollectionGames {
			c.CollectionGames = append(c.CollectionGames, cg)
		}
		c.CollectionGames = nil

		profile.ProfileCollections = append(profile.ProfileCollections, &models.ProfileCollection{
			// Other fields are set when saving the association
			Collection: c,
			Position:   int64(i),
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSave(conn, &hades.SaveParams{
			Record: profile,
			Assocs: []string{"ProfileCollections"},
			DontCull: []interface{}{
				&itchio.CollectionGame{},
			},
		})
	})

	err = sendDBCollections()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.FetchProfileCollectionsResult{}
	return res, nil
}
