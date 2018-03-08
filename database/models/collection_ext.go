package models

import (
	"github.com/itchio/butler/database/hades"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

// Collection is defined in `go-itchio`, but helper functions are here

func CollectionByID(db *gorm.DB, id int64) *itchio.Collection {
	var c itchio.Collection
	req := db.Where("id = ?", id).First(&c)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil
		}
		panic(req.Error)
	}
	return &c
}

type collectionExt struct {
	*itchio.Collection
}

func CollectionExt(c *itchio.Collection) collectionExt {
	return collectionExt{
		Collection: c,
	}
}

func (ce collectionExt) PreloadCollectionGames(db *gorm.DB) {
	MustPreload(db, &hades.PreloadParams{
		Record: ce.Collection,
		Fields: []hades.PreloadField{
			hades.PreloadField{Name: "CollectionGames", OrderBy: `"position" ASC`},
			hades.PreloadField{Name: "CollectionGames.Game"},
		},
	})
}
