package search

import (
	"fmt"
	"log"

	"crawshaw.io/sqlite"
	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/hades"
	"github.com/itchio/httpkit/neterr"
	"github.com/pkg/errors"
	"xorm.io/builder"
)

func SearchUsers(rc *butlerd.RequestContext, params butlerd.SearchUsersParams) (*butlerd.SearchUsersResult, error) {
	if params.Query == "" {
		// return empty users set
		return &butlerd.SearchUsersResult{
			Users: nil,
		}, nil
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
	// perform local search
	//----------------------------------

	doLocalSearch()

	//----------------------------------
	// perform API request
	//----------------------------------

	_, client := rc.ProfileClient(params.ProfileID)
	searchRes, err := client.SearchUsers(rc.Ctx, itchio.SearchUsersParams{
		Query: params.Query,
		Page:  1,
	})
	if err != nil {
		if neterr.IsNetworkError(err) {
			log.Printf("Seemingly offline, returning local results only")
			return &butlerd.SearchUsersResult{
				Users: users,
			}, nil
		}

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

	res := &butlerd.SearchUsersResult{
		Users: users,
	}
	return res, nil
}
