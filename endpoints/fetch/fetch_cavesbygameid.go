package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func FetchCavesByGameID(rc *buse.RequestContext, params *buse.FetchCavesByGameIDParams) (*buse.FetchCavesByGameIDResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	caves, err := models.CavesByGameID(db, params.GameID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = PreloadCaves(db, rc.Consumer, caves)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var formattedCaves []*buse.Cave
	for _, c := range caves {
		formattedCaves = append(formattedCaves, formatCave(c))
	}

	res := &buse.FetchCavesByGameIDResult{
		Caves: formattedCaves,
	}
	return res, nil
}
