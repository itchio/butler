package collections

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func Register(router *butlerd.Router) {
	messages.CollectionsCreate.Register(router, Create)
	messages.CollectionsUpdate.Register(router, Update)
	messages.CollectionsDelete.Register(router, Delete)
	messages.CollectionsAddGame.Register(router, AddGame)
	messages.CollectionsRemoveGame.Register(router, RemoveGame)
	messages.CollectionsUpdateGame.Register(router, UpdateGame)
	messages.CollectionsOrderGames.Register(router, OrderGames)
}

func Create(rc *butlerd.RequestContext, params butlerd.CollectionsCreateParams) (*butlerd.CollectionsCreateResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	res, err := client.CreateCollection(rc.Ctx, itchio.CreateCollectionParams{
		Title:       params.Title,
		Private:     params.Private,
		Description: params.Description,
		Layout:      params.Layout,
		GameID:      params.GameID,
		Blurb:       params.Blurb,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if res.Collection == nil {
		return nil, errors.New("API returned no collection")
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		saveCollection(conn, res.Collection)
		models.FetchTargetForProfileCollections(profile.ID).MustExpire(conn)
	})

	return &butlerd.CollectionsCreateResult{
		Collection: res.Collection,
	}, nil
}

func Update(rc *butlerd.RequestContext, params butlerd.CollectionsUpdateParams) (*butlerd.CollectionsUpdateResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	res, err := client.UpdateCollection(rc.Ctx, itchio.UpdateCollectionParams{
		CollectionID: params.CollectionID,
		Title:        params.Title,
		Description:  params.Description,
		Private:      params.Private,
		Layout:       params.Layout,
		OnProfile:    params.OnProfile,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if res.Collection == nil {
		return nil, errors.New("API returned no collection")
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		saveCollection(conn, res.Collection)
		// on_profile changes which collections are listed, and the
		// title shows up in every admin's list
		expireProfileCollectionLists(conn, profile.ID, params.CollectionID)
	})

	return &butlerd.CollectionsUpdateResult{
		Collection: res.Collection,
	}, nil
}

func Delete(rc *butlerd.RequestContext, params butlerd.CollectionsDeleteParams) (*butlerd.CollectionsDeleteResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	_, err := client.DeleteCollection(rc.Ctx, itchio.DeleteCollectionParams{
		CollectionID: params.CollectionID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		expireProfileCollectionLists(conn, profile.ID, params.CollectionID)
		forgetCollection(conn, params.CollectionID)
	})

	return &butlerd.CollectionsDeleteResult{}, nil
}

func AddGame(rc *butlerd.RequestContext, params butlerd.CollectionsAddGameParams) (*butlerd.CollectionsAddGameResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	res, err := client.AddCollectionGame(rc.Ctx, itchio.AddCollectionGameParams{
		CollectionID: params.CollectionID,
		GameID:       params.GameID,
		Blurb:        params.Blurb,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if res.CollectionGame == nil {
		return nil, errors.New("API returned no collection game")
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		saveCollectionGame(conn, res.CollectionGame)
		expireCollectionContents(conn, profile.ID, params.CollectionID)
	})

	return &butlerd.CollectionsAddGameResult{
		CollectionGame: res.CollectionGame,
	}, nil
}

func RemoveGame(rc *butlerd.RequestContext, params butlerd.CollectionsRemoveGameParams) (*butlerd.CollectionsRemoveGameResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	res, err := client.RemoveCollectionGame(rc.Ctx, itchio.RemoveCollectionGameParams{
		CollectionID: params.CollectionID,
		GameID:       params.GameID,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustDelete(conn, &itchio.CollectionGame{}, builder.Eq{
			"collection_id": params.CollectionID,
			"game_id":       params.GameID,
		})
		expireCollectionContents(conn, profile.ID, params.CollectionID)
	})

	return &butlerd.CollectionsRemoveGameResult{
		Removed: res.Removed,
	}, nil
}

func UpdateGame(rc *butlerd.RequestContext, params butlerd.CollectionsUpdateGameParams) (*butlerd.CollectionsUpdateGameResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	res, err := client.UpdateCollectionGame(rc.Ctx, itchio.UpdateCollectionGameParams{
		CollectionID: params.CollectionID,
		GameID:       params.GameID,
		Blurb:        params.Blurb,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if res.CollectionGame == nil {
		return nil, errors.New("API returned no collection game")
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		saveCollectionGame(conn, res.CollectionGame)
		expireCollectionContents(conn, profile.ID, params.CollectionID)
	})

	return &butlerd.CollectionsUpdateGameResult{
		CollectionGame: res.CollectionGame,
	}, nil
}

func OrderGames(rc *butlerd.RequestContext, params butlerd.CollectionsOrderGamesParams) (*butlerd.CollectionsOrderGamesResult, error) {
	profile, client := rc.ProfileClient(params.ProfileID)

	_, err := client.OrderCollectionGames(rc.Ctx, itchio.OrderCollectionGamesParams{
		CollectionID:  params.CollectionID,
		GameIDs:       params.GameIDs,
		RemoveGameIDs: params.RemoveGameIDs,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rc.WithConn(func(conn *sqlite.Conn) {
		if len(params.RemoveGameIDs) > 0 {
			models.MustDelete(conn, &itchio.CollectionGame{}, builder.And(
				builder.Eq{"collection_id": params.CollectionID},
				builder.In("game_id", params.RemoveGameIDs),
			))
		}
		expireCollectionContents(conn, profile.ID, params.CollectionID)
	})

	return &butlerd.CollectionsOrderGamesResult{}, nil
}

// saveCollection stores the collection row returned by an edit so
// local reads reflect it right away. Games are not part of the
// payload and are left to Fetch.Collection.Games. HasGame is never
// persisted (hades ignores it in go-itchio).
func saveCollection(conn *sqlite.Conn, collection *itchio.Collection) {
	collection.CollectionGames = nil
	models.MustSave(conn, collection)
}

// saveCollectionGame stores a single membership row, along with its
// game when the API included it.
func saveCollectionGame(conn *sqlite.Conn, cg *itchio.CollectionGame) {
	if cg.Game != nil {
		models.MustSave(conn, cg, hades.Assoc("Game"))
	} else {
		models.MustSave(conn, cg)
	}
}

// forgetCollection removes every local trace of a deleted collection.
func forgetCollection(conn *sqlite.Conn, collectionID int64) {
	models.MustDelete(conn, &itchio.CollectionGame{}, builder.Eq{"collection_id": collectionID})
	models.MustDelete(conn, &models.ProfileCollection{}, builder.Eq{"collection_id": collectionID})
	models.MustDelete(conn, &itchio.Collection{}, builder.Eq{"id": collectionID})
	models.FetchTargetForCollectionGames(collectionID).MustExpire(conn)
	models.FetchTargetForCollection(collectionID).MustExpire(conn)
}

// expireCollectionContents marks everything that depends on a
// collection's membership as stale: the games list, the collection
// itself (gamesCount, updatedAt), and the collection lists that
// carry the same counters.
func expireCollectionContents(conn *sqlite.Conn, profileID int64, collectionID int64) {
	models.FetchTargetForCollectionGames(collectionID).MustExpire(conn)
	models.FetchTargetForCollection(collectionID).MustExpire(conn)
	expireProfileCollectionLists(conn, profileID, collectionID)
}

// expireProfileCollectionLists expires the collection list of the
// acting profile and of every other local profile that lists the
// collection: a collection can be shared between its owner and
// its admins.
func expireProfileCollectionLists(conn *sqlite.Conn, profileID int64, collectionID int64) {
	models.FetchTargetForProfileCollections(profileID).MustExpire(conn)

	var rows []*models.ProfileCollection
	models.MustSelect(conn, &rows, builder.Eq{"collection_id": collectionID}, hades.Search{})
	for _, row := range rows {
		if row.ProfileID != profileID {
			models.FetchTargetForProfileCollections(row.ProfileID).MustExpire(conn)
		}
	}
}
