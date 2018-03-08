package models

import (
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

// Game is defined in `go-itchio`, but helper functions are here

func GameByID(db *gorm.DB, id int64) *itchio.Game {
	var g itchio.Game
	req := db.Where("id = ?", id).First(&g)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil
		}
		panic(req.Error)
	}
	return &g
}
