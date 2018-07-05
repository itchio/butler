package meta

import (
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
)

func Register(router *butlerd.Router) {
	messages.MetaAuthenticate.Register(router, func(rc *butlerd.RequestContext, params butlerd.MetaAuthenticateParams) (*butlerd.MetaAuthenticateResult, error) {
		return nil, errors.Errorf("Meta.Authenticate not needed (and not valid) for your current transport")
	})
}
