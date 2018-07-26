package meta

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/pkg/errors"
)

var establishedAt *time.Time
var establishedLock sync.Mutex

func Register(router *butlerd.Router) {
	messages.MetaAuthenticate.Register(router, func(rc *butlerd.RequestContext, params butlerd.MetaAuthenticateParams) (*butlerd.MetaAuthenticateResult, error) {
		return nil, errors.Errorf("Meta.Authenticate not needed (and not valid) for your current transport")
	})
	messages.MetaFlow.Register(router, func(rc *butlerd.RequestContext, params butlerd.MetaFlowParams) (*butlerd.MetaFlowResult, error) {
		establishedLock.Lock()
		if establishedAt == nil {
			now := time.Now().UTC()
			establishedAt = &now
			establishedLock.Unlock()
		} else {
			lastEstablished := *establishedAt
			establishedLock.Unlock()
			return nil, errors.Errorf("Cannot establish Meta.Flow twice in the same daemon instance. Last established %s", lastEstablished)
		}

		messages.MetaFlowEstablished.Notify(rc, butlerd.MetaFlowEstablishedNotification{
			PID: int64(os.Getpid()),
		})

		var never chan struct{}
		select {
		case <-never: // blocks forever
		case <-rc.Ctx.Done():
			log.Printf("Meta.Flow cancelled, requesting shutdown")
			rc.Shutdown()
		}
		return &butlerd.MetaFlowResult{}, nil
	})
}
