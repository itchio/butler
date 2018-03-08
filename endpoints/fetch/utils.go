package fetch

import (
	"github.com/itchio/butler/buse"
	"github.com/itchio/butler/database/hades"
	"github.com/itchio/butler/database/models"
)

func HadesContext(rc *buse.RequestContext) *hades.Context {
	return hades.NewContext(rc.DB(), models.AllModels, rc.Consumer)
}
