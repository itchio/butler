package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func FetchCaves(rc *butlerd.RequestContext, params *butlerd.FetchCavesParams) (*butlerd.FetchCavesResult, error) {
	var caves []*models.Cave
	err := rc.DB().Find(&caves).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	models.PreloadCaves(rc.DB(), caves)

	var formattedCaves []*butlerd.Cave
	for _, cave := range caves {
		formattedCaves = append(formattedCaves, FormatCave(rc.DB(), cave))
	}

	res := &butlerd.FetchCavesResult{
		Caves: formattedCaves,
	}
	return res, nil
}
