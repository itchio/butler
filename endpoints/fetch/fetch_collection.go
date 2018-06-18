package fetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch/lazyfetch"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func FetchCollection(rc *butlerd.RequestContext, params butlerd.FetchCollectionParams) (*butlerd.FetchCollectionResult, error) {
	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	ft := models.FetchTargetForCollection(params.CollectionID)
	res := &butlerd.FetchCollectionResult{}

	lazyfetch.Do(rc, ft, params, res, func(targets lazyfetch.Targets) {
		_, client := rc.ProfileClient(params.ProfileID)

		collRes, err := client.GetCollection(itchio.GetCollectionParams{
			CollectionID: params.CollectionID,
		})
		models.Must(err)

		collection := collRes.Collection
		collection.CollectionGames = nil

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, collRes.Collection)
		})
	})

	rc.WithConn(func(conn *sqlite.Conn) {
		res.Collection = models.CollectionByID(conn, params.CollectionID)
	})

	if res.Collection == nil && !params.Fresh {
		params.Fresh = true
		return FetchCollection(rc, params)
	}

	return res, nil
}
