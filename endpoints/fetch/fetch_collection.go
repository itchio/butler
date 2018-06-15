package fetch

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func FetchCollection(rc *butlerd.RequestContext, params *butlerd.FetchCollectionParams) (*butlerd.FetchCollectionResult, error) {
	if params.CollectionID == 0 {
		return nil, errors.New("collectionId must be non-zero")
	}

	consumer := rc.Consumer
	ft := models.FetchTarget{
		Type: "collection",
		ID:   params.CollectionID,
		TTL:  10 * time.Minute,
	}

	fresh := false
	res := &butlerd.FetchCollectionResult{}

	if params.Fresh {
		consumer.Infof("Doing remote fetch (Fresh specified)")
		fresh = true
	} else if rc.WithConnBool(ft.MustIsStale) {
		consumer.Infof("Returning stale info")
		res.Stale = true
	}

	if fresh {
		_, client := rc.ProfileClient(params.ProfileID)

		consumer.Debugf("Querying API...")
		collRes, err := client.GetCollection(itchio.GetCollectionParams{
			CollectionID: params.CollectionID,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		collection := collRes.Collection
		collection.CollectionGames = nil

		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSave(conn, collRes.Collection)
			ft.MustMarkFresh(conn)
		})
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		res.Collection = models.CollectionByID(conn, params.CollectionID)
	})

	if res.Collection == nil && !params.Fresh {
		freshParams := *params
		freshParams.Fresh = true
		return FetchCollection(rc, &freshParams)
	}

	return res, nil
}
