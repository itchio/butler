package install

import (
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

func InstallLocationsGetByID(rc *butlerd.RequestContext, params butlerd.InstallLocationsGetByIDParams) (*butlerd.InstallLocationsGetByIDResult, error) {
	if params.ID == "" {
		return nil, errors.Errorf("id must be set")
	}

	conn := rc.GetConn()
	defer rc.PutConn(conn)

	il := models.InstallLocationByID(conn, params.ID)
	if il == nil {
		return nil, errors.Errorf("install location (%s) not found", params.ID)
	}

	res := &butlerd.InstallLocationsGetByIDResult{
		InstallLocation: fetch.FormatInstallLocation(conn, rc.Consumer, il),
	}
	return res, nil
}

func InstallLocationsList(rc *butlerd.RequestContext, params butlerd.InstallLocationsListParams) (*butlerd.InstallLocationsListResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)

	var locations []*models.InstallLocation
	models.MustSelect(conn, &locations, builder.NewCond(), hades.Search{})

	var flocs []*butlerd.InstallLocationSummary
	for _, il := range locations {
		flocs = append(flocs, fetch.FormatInstallLocation(conn, rc.Consumer, il))
	}

	res := &butlerd.InstallLocationsListResult{
		InstallLocations: flocs,
	}
	return res, nil
}

func InstallLocationsAdd(rc *butlerd.RequestContext, params butlerd.InstallLocationsAddParams) (*butlerd.InstallLocationsAddResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)
	consumer := rc.Consumer

	hadID := false
	if params.ID == "" {
		hadID = true
		freshUuid, err := uuid.NewV4()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		params.ID = freshUuid.String()
	}
	if params.Path == "" {
		return nil, errors.New("path must be set")
	}

	if hadID {
		existing := models.InstallLocationByID(conn, params.ID)
		if existing != nil {
			if existing.Path == params.Path {
				consumer.Statf("(%s) exists, and has same path (%s), doing nothing", params.ID, params.Path)
				res := &butlerd.InstallLocationsAddResult{}
				return res, nil
			}
			return nil, errors.Errorf("(%s) exists but has path (%s) - we were passed (%s)", params.ID, existing.Path, params.Path)
		}
	}

	il := &models.InstallLocation{
		ID:   params.ID,
		Path: params.Path,
	}
	models.MustSave(conn, il)

	res := &butlerd.InstallLocationsAddResult{}
	return res, nil
}

func InstallLocationsRemove(rc *butlerd.RequestContext, params butlerd.InstallLocationsRemoveParams) (*butlerd.InstallLocationsRemoveResult, error) {
	conn := rc.GetConn()
	defer rc.PutConn(conn)
	consumer := rc.Consumer

	if params.ID == "" {
		return nil, errors.Errorf("id must be set")
	}

	il := models.InstallLocationByID(conn, params.ID)
	if il == nil {
		consumer.Statf("Install location (%s) does not exist, doing nothing")
		res := &butlerd.InstallLocationsRemoveResult{}
		return res, nil
	}

	caveCount := models.MustCount(conn, &models.Cave{}, builder.Eq{"install_location_id": il.ID})
	if caveCount > 0 {
		// TODO: suggest moving to another install location
		return nil, errors.Errorf("Refusing to remove install location (%s) because it is not empty", params.ID)
	}

	locationCount := models.MustCount(conn, &models.InstallLocation{}, builder.NewCond())
	if locationCount <= 1 {
		return nil, errors.Errorf("Refusing to remove last install location")
	}

	models.MustDelete(conn, &models.InstallLocation{}, builder.Eq{"id": il.ID})
	res := &butlerd.InstallLocationsRemoveResult{}
	return res, nil
}
