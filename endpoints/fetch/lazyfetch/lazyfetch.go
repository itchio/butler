package lazyfetch

import (
	"time"

	"github.com/itchio/butler/butlerd/horror"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

type ProfiledLazyFetchParams interface {
	ProfiledParams
	LazyFetchParams
}

type ProfiledParams interface {
	GetProfileID() int64
}

type LazyFetchParams interface {
	IsFresh() bool
}

type LazyFetchResponse interface {
	SetStale(stale bool)
}

type Targets interface {
	Add(ft models.FetchTarget)
}

type Task func(t Targets)

func Do(
	rc *butlerd.RequestContext,
	ft models.FetchTarget,
	params LazyFetchParams,
	res LazyFetchResponse,
	task Task) {

	if params.IsFresh() {
		rc.Consumer.Infof("Fetching fresh data...")
		startTime := time.Now()
		_, err, shared := rc.Group.Do(ft.Key(), func() (res interface{}, err error) {
			// we have to recover from panics here, otherwise
			// we might be stuck with a singleflight.Do forever
			defer horror.RecoverInto(&err)

			ts := &targets{
				items: []models.FetchTarget{ft},
			}
			task(ts)
			rc.WithConn(func(conn *sqlite.Conn) {
				models.MustMarkAllFresh(conn, ts.items)
			})
			return
		})
		if err != nil {
			panic(err)
		}

		if shared {
			rc.Consumer.Infof("Waited %s for fetch (shared with another call)", time.Since(startTime))
		} else {
			rc.Consumer.Infof("Waited %s for fetch (non-shared)", time.Since(startTime))
		}
	} else if rc.WithConnBool(ft.MustIsStale) {
		res.SetStale(true)
	}
}

//

type targets struct {
	items []models.FetchTarget
}

func (ts *targets) Add(ft models.FetchTarget) {
	ts.items = append(ts.items, ft)
}
