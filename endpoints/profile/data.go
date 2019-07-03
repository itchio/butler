package profile

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

func DataPut(rc *butlerd.RequestContext, params butlerd.ProfileDataPutParams) (*butlerd.ProfileDataPutResult, error) {
	// will panic if invalid profile or missing param
	rc.ProfileClient(params.ProfileID)

	pd := &models.ProfileData{
		ProfileID: params.ProfileID,
		Key:       params.Key,
		Value:     params.Value,
	}
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSave(conn, pd)
	})

	res := &butlerd.ProfileDataPutResult{}
	return res, nil
}

func DataGet(rc *butlerd.RequestContext, params butlerd.ProfileDataGetParams) (*butlerd.ProfileDataGetResult, error) {
	// will panic if invalid profile or missing param
	rc.ProfileClient(params.ProfileID)

	var ok bool
	var pd models.ProfileData
	rc.WithConn(func(conn *sqlite.Conn) {
		ok = models.MustSelectOne(conn, &pd,
			builder.Eq{
				"profile_id": params.ProfileID,
				"key":        params.Key,
			},
		)
	})

	res := &butlerd.ProfileDataGetResult{
		OK:    ok,
		Value: pd.Value,
	}
	return res, nil
}
