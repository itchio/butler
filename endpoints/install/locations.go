package install

import (
	"github.com/go-errors/errors"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/butler/endpoints/fetch"
	uuid "github.com/satori/go.uuid"
)

func InstallLocationsGetByID(rc *butlerd.RequestContext, params *butlerd.InstallLocationsGetByIDParams) (*butlerd.InstallLocationsGetByIDResult, error) {
	if params.ID == "" {
		return nil, errors.Errorf("id must be set")
	}

	il := models.InstallLocationByID(rc.DB(), params.ID)
	if il == nil {
		return nil, errors.Errorf("install location (%s) not found", params.ID)
	}

	res := &butlerd.InstallLocationsGetByIDResult{
		InstallLocation: fetch.FormatInstallLocation(rc, il),
	}
	return res, nil
}

func InstallLocationsList(rc *butlerd.RequestContext, params *butlerd.InstallLocationsListParams) (*butlerd.InstallLocationsListResult, error) {
	var locations []*models.InstallLocation
	err := rc.DB().Find(&locations).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	var flocs []*butlerd.InstallLocationSummary
	for _, il := range locations {
		flocs = append(flocs, fetch.FormatInstallLocation(rc, il))
	}

	res := &butlerd.InstallLocationsListResult{
		InstallLocations: flocs,
	}
	return res, nil
}

func InstallLocationsAdd(rc *butlerd.RequestContext, params *butlerd.InstallLocationsAddParams) (*butlerd.InstallLocationsAddResult, error) {
	consumer := rc.Consumer

	hadID := false
	if params.ID == "" {
		hadID = true
		freshUuid, err := uuid.NewV4()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		params.ID = freshUuid.String()
	}
	if params.Path == "" {
		return nil, errors.New("path must be set")
	}

	if hadID {
		existing := models.InstallLocationByID(rc.DB(), params.ID)
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
	err := rc.DB().Save(il).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &butlerd.InstallLocationsAddResult{}
	return res, nil
}

func InstallLocationsRemove(rc *butlerd.RequestContext, params *butlerd.InstallLocationsRemoveParams) (*butlerd.InstallLocationsRemoveResult, error) {
	consumer := rc.Consumer

	if params.ID == "" {
		return nil, errors.Errorf("id must be set")
	}

	il := models.InstallLocationByID(rc.DB(), params.ID)
	if il == nil {
		consumer.Statf("Install location (%s) does not exist, doing nothing")
		res := &butlerd.InstallLocationsRemoveResult{}
		return res, nil
	}

	var caveCount int64
	err := rc.DB().Model(&models.Cave{}).Where("install_location_id = ?", il.ID).Count(&caveCount).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if caveCount > 0 {
		// TODO: suggest moving to another install location
		return nil, errors.Errorf("Refusing to remove install location (%s) because it is not empty", params.ID)
	}

	var locationCount int64
	err = rc.DB().Model(&models.InstallLocation{}).Count(&locationCount).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	if locationCount == 1 {
		return nil, errors.Errorf("Refusing to remove last install location")
	}

	err = rc.DB().Delete(il).Error
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	res := &butlerd.InstallLocationsRemoveResult{}
	return res, nil
}
