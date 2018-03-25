package downloads

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
)

func HadesContext(rc *butlerd.RequestContext) *hades.Context {
	return hades.NewContext(rc.DB(), models.AllModels, rc.Consumer)
}
