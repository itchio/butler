package fetch

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/pkg/errors"
)

func FetchCaves(rc *butlerd.RequestContext, params *butlerd.FetchCavesParams) (*butlerd.FetchCavesResult, error) {
	var caves []*models.Cave
	err := rc.DB().Find(&caves).Error
	if err != nil {
		return nil, errors.WithStack(err)
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
