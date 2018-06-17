package lazyfetch

import (
	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
)

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
		ts := &targets{
			items: []models.FetchTarget{ft},
		}
		task(ts)
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustMarkAllFresh(conn, ts.items)
		})
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
