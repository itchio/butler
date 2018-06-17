package search

import (
	"fmt"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/pkg/errors"
)

func SearchUsers(rc *butlerd.RequestContext, params *butlerd.SearchUsersParams) (*butlerd.SearchUsersResult, error) {
	if params.Query == "" {
		// return empty users set
		err := messages.SearchUsersYield.Notify(rc, &butlerd.SearchUsersYieldNotification{
			Users: nil,
		})
		if err != nil {
			return nil, errors.WithStack(err)
		}

		return &butlerd.SearchUsersResult{}, nil
	}

	var users []*itchio.User

	doLocalSearch := func() {
		users = nil
		q := fmt.Sprintf("%%%s%%", params.Query)
		rc.WithConn(func(conn *sqlite.Conn) {
			models.MustSelect(conn, &users,
				builder.Or(
					builder.Like{"lower(display_name)", q},
					builder.Like{"lower(username)", q},
				),
				hades.Search{}.Limit(4),
			)
		})
	}

	//----------------------------------
	// return results from local DB
	//----------------------------------

	doLocalSearch()
	err := messages.SearchUsersYield.Notify(rc, &butlerd.SearchUsersYieldNotification{
		Users: users,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	//----------------------------------
	// perform API request
	//----------------------------------

	_, client := rc.ProfileClient(params.ProfileID)
	searchRes, err := client.SearchUsers(itchio.SearchUsersParams{
		Query: params.Query,
		Page:  1,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	//----------------------------------
	// save remote results which were already in local cache
	//----------------------------------

	localMap := make(map[int64]bool)
	for _, u := range users {
		localMap[u.ID] = true
	}

	var updatedUsers []*itchio.User
	for _, u := range searchRes.Users {
		if localMap[u.ID] {
			updatedUsers = append(updatedUsers, u)
		}
	}
	rc.WithConn(func(conn *sqlite.Conn) {
		models.MustSave(conn, updatedUsers)
	})

	//----------------------------------
	// send local + remote results
	//----------------------------------

	doLocalSearch()
	for _, u := range searchRes.Users {
		if len(users) > 15 {
			break
		}

		if _, ok := localMap[u.ID]; !ok {
			users = append(users, u)
		}
	}

	err = messages.SearchUsersYield.Notify(rc, &butlerd.SearchUsersYieldNotification{
		Users: users,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.SearchUsersResult{}
	return res, nil
}
