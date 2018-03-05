package models

import (
	"github.com/go-errors/errors"
	itchio "github.com/itchio/go-itchio"
	"github.com/jinzhu/gorm"
)

// Collection is defined in `go-itchio`, but helper functions are here

func CollectionByID(db *gorm.DB, id int64) (*itchio.Collection, error) {
	c := &itchio.Collection{}
	req := db.Where("id = ?", id).First(c)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil, nil
		}
		return nil, errors.Wrap(req.Error, 0)
	}
	return c, nil
}
