package fetch

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func FetchCave(rc *buse.RequestContext, params *buse.FetchCaveParams) (*buse.FetchCaveResult, error) {
	db, err := rc.DB()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	cave, err := models.CaveByID(db, params.CaveID)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.FetchCaveResult{
		Cave: cave,
	}
	return res, nil
}
