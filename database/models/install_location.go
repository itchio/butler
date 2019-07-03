package models

import (
	"path/filepath"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
)

type InstallLocation struct {
	ID string `json:"id" hades:"primary_key"`

	Path string `json:"path"`

	Caves []*Cave `json:"caves"`
}

func InstallLocationByID(conn *sqlite.Conn, id string) *InstallLocation {
	var il InstallLocation
	if MustSelectOne(conn, &il, builder.Eq{"id": id}) {
		return &il
	}
	return nil
}

func (il *InstallLocation) GetInstallFolder(folderName string) string {
	return filepath.Join(il.Path, folderName)
}

func (il *InstallLocation) GetStagingFolder(installID string) string {
	return filepath.Join(il.Path, "downloads", installID)
}

func (il *InstallLocation) GetCaves(conn *sqlite.Conn) []*Cave {
	MustPreload(conn, il,
		hades.Assoc("Caves"),
	)
	return il.Caves
}
