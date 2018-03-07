package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
)

// Preload Game, Upload and Build for a given cave
func PreloadCaves(db *gorm.DB, consumer *state.Consumer, caveOrCaves interface{}) error {
	err := hades.NewContext(db, consumer).Preload(db, &hades.PreloadParams{
		Record: caveOrCaves,
		Fields: []hades.PreloadField{
			hades.PreloadField{Name: "Game"},
			hades.PreloadField{Name: "Upload"},
			hades.PreloadField{Name: "Build"},
		},
	})
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
