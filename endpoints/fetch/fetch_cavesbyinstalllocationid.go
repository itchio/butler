package fetch

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func FetchCavesByInstallLocationID(rc *buse.RequestContext, params *buse.FetchCavesByInstallLocationIDParams) (*buse.FetchCavesByInstallLocationIDResult, error) {
	caves := models.CavesByInstallLocationID(rc.DB(), params.InstallLocationID)
	models.PreloadCaves(rc.DB(), caves)

	var formattedCaves []*buse.Cave
	for _, c := range caves {
		formattedCaves = append(formattedCaves, formatCave(rc.DB(), c))
	}

	res := &buse.FetchCavesByInstallLocationIDResult{
		Caves: formattedCaves,
	}
	return res, nil
}
