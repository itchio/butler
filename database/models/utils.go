package models

import (
	"github.com/itchio/butler/database/hades"
	"github.com/jinzhu/gorm"
)

func HadesContext(db *gorm.DB) *hades.Context {
	return hades.NewContext(db, AllModels, nil)
}

func Preload(db *gorm.DB, params *hades.PreloadParams) error {
	return HadesContext(db).Preload(db, params)
}

func MustPreload(db *gorm.DB, params *hades.PreloadParams) error {
	err := Preload(db, params)
	if err != nil {
		panic(err)
	}
	return nil
}

func PreloadSimple(db *gorm.DB, record interface{}, fields ...string) error {
	var pfs []hades.PreloadField
	for _, f := range fields {
		pfs = append(pfs, hades.PreloadField{
			Name: f,
		})
	}

	return Preload(db, &hades.PreloadParams{
		Record: record,
		Fields: pfs,
	})
}

func MustPreloadSimple(db *gorm.DB, record interface{}, fields ...string) {
	err := PreloadSimple(db, record, fields...)
	if err != nil {
		panic(err)
	}
}
