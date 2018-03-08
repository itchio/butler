package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func FetchCaves(rc *buse.RequestContext, params *buse.FetchCavesParams) (*buse.FetchCavesResult, error) {
	var caves []*models.Cave
	err := rc.DB().Find(&caves).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	models.PreloadCaves(rc.DB(), caves)

	var formattedCaves []*buse.Cave
	for _, cave := range caves {
		formattedCaves = append(formattedCaves, formatCave(rc.DB(), cave))
	}

	res := &buse.FetchCavesResult{
		Caves: formattedCaves,
	}
	return res, nil
}
