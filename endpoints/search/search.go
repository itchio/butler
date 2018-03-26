package search

import (
	"fmt"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/butlerd/messages"
	itchio "github.com/itchio/go-itchio"
	"github.com/pkg/errors"
)

func Register(router *butlerd.Router) {
	messages.SearchGames.Register(router, SearchGames)
	messages.SearchUsers.Register(router, SearchUsers)
}

func SearchGames(rc *butlerd.RequestContext, params *butlerd.SearchGamesParams) (*butlerd.SearchGamesResult, error) {
	var games []*itchio.Game
	db := rc.DB()
	q := fmt.Sprintf("%%%s%%", params.Query)
	err := db.Where("lower(title) like ?", q).Limit(4).Find(&games).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = messages.SearchGamesYield.Notify(rc, &butlerd.SearchGamesYieldNotification{
		Games: games,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, client := rc.ProfileClient(params.ProfileID)

	searchRes, err := client.SearchGames(&itchio.SearchGamesParams{
		Query: params.Query,
		Page:  1,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// TODO: rank results by similarity
	localMap := make(map[int64]bool)
	for _, g := range games {
		localMap[g.ID] = true
	}

	for _, g := range searchRes.Games {
		if len(games) > 15 {
			break
		}

		if _, ok := localMap[g.ID]; !ok {
			games = append(games, g)
		}
	}

	err = messages.SearchGamesYield.Notify(rc, &butlerd.SearchGamesYieldNotification{
		Games: games,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := &butlerd.SearchGamesResult{}
	return res, nil
}

func SearchUsers(rc *butlerd.RequestContext, params *butlerd.SearchUsersParams) (*butlerd.SearchUsersResult, error) {
	var users []*itchio.User
	db := rc.DB()
	q := fmt.Sprintf("%%%s%%", params.Query)
	err := db.Where("lower(display_name) like ? OR lower(username) like ?", q, q).Limit(4).Find(&users).Error

	if err != nil {
		return nil, errors.WithStack(err)
	}

	err = messages.SearchUsersYield.Notify(rc, &butlerd.SearchUsersYieldNotification{
		Users: users,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	_, client := rc.ProfileClient(params.ProfileID)

	searchRes, err := client.SearchUsers(&itchio.SearchUsersParams{
		Query: params.Query,
		Page:  1,
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// TODO: rank results by similarity
	localMap := make(map[int64]bool)
	for _, u := range users {
		localMap[u.ID] = true
	}

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
