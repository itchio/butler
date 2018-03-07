package models

import (
	"path/filepath"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

type InstallLocation struct {
	ID string `json:"id" gorm:"primary_key"`

	Path string `json:"path"`
}

func InstallLocationByID(db *gorm.DB, id string) (*InstallLocation, error) {
	il := &InstallLocation{}
	req := db.Where("id = ?", id).First(il)
	if req.Error != nil {
		if req.RecordNotFound() {
			return nil, nil
		}
		return nil, errors.Wrap(req.Error, 0)
	}
	return il, nil
}

func (il *InstallLocation) AbsoluteFolderPath(folderName string) string {
	return filepath.Join(il.Path, folderName)
}
