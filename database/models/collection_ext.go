package models

import (
	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
)

// Collection is defined in `go-itchio`, but helper functions are here

func CollectionByID(conn *sqlite.Conn, id int64) *itchio.Collection {
	var c itchio.Collection
	if MustSelectOne(conn, &c, builder.Eq{"id": id}) {
		return &c
	}
	return nil
}

type collectionExt struct {
	*itchio.Collection
}

func CollectionExt(c *itchio.Collection) collectionExt {
	return collectionExt{
		Collection: c,
	}
}

func (ce collectionExt) PreloadCollectionGames(conn *sqlite.Conn) {
	MustPreload(conn, &hades.PreloadParams{
		Record: ce.Collection,
		Fields: []hades.PreloadField{
			{Name: "CollectionGames", Search: hades.Search().OrderBy("position ASC")},
			{Name: "CollectionGames.Game"},
		},
	})
}
