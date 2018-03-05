package models

import (
	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

// Game is defined in `go-itchio`, but helper functions are here

func GameByID(db *gorm.DB, id int64) (*itchio.Game, error) {
	g := &itchio.Game{}
	req := db.Where("id = ?", id).First(g)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil, nil
		}
		return nil, errors.Wrap(req.Error, 0)
	}
	return g, nil
}
