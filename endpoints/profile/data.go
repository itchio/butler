package profile

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/models"
)

func DataPut(rc *buse.RequestContext, params *buse.ProfileDataPutParams) (*buse.ProfileDataPutResult, error) {
	// will panic if invalid profile or missing param
	rc.ProfileClient(params.ProfileID)

	db := rc.DB()

	pd := &models.ProfileData{
		ProfileID: params.ProfileID,
		Key:       params.Key,
		Value:     params.Value,
	}
	err := db.Save(&pd).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &buse.ProfileDataPutResult{}
	return res, nil
}

func DataGet(rc *buse.RequestContext, params *buse.ProfileDataGetParams) (*buse.ProfileDataGetResult, error) {
	// will panic if invalid profile or missing param
	rc.ProfileClient(params.ProfileID)

	db := rc.DB()

	pd := &models.ProfileData{}
	req := db.Where("profile_id = ? AND key = ?", params.ProfileID, params.Key).Find(pd)
	if req.Error != nil {
		if req.RecordNotFound() {
			res := &buse.ProfileDataGetResult{
				OK: false,
			}
			return res, nil
		}
		return nil, errors.Wrap(req.Error, 0)
	}

	res := &buse.ProfileDataGetResult{
		OK:    true,
		Value: pd.Value,
	}
	return res, nil
}
