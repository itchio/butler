package models

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
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
	MustPreload(conn, ce.Collection,
		hades.AssocWithSearch("CollectionGames", hades.Search{}.OrderBy("position ASC"),
			hades.Assoc("Game"),
		),
	)
}
