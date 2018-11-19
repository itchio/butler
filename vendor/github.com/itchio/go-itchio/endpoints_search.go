package itchio

// SearchGamesParams : params for SearchGames
type SearchGamesParams struct {
	Query string
	Page  int64
}

// SearchGamesResponse : response for SearchGames
type SearchGamesResponse struct {
	Page    int64   `json:"page"`
	PerPage int64   `json:"perPage"`
	Games   []*Game `json:"games"`
}

// SearchGames performs a text search for games (or any project type).
// The games must be published, and not deindexed. There are a bunch
// of subtleties about visibility and ranking, but that's internal.
func (c *Client) SearchGames(params SearchGamesParams) (*SearchGamesResponse, error) {
	q := NewQuery(c, "/search/games")
	q.AddString("query", params.Query)
	q.AddInt64IfNonZero("page", params.Page)
	r := &SearchGamesResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// SearchUsersParams : params for SearchUsers
type SearchUsersParams struct {
	Query string
	Page  int64
}

// SearchUsersResponse : response for SearchUsers
type SearchUsersResponse struct {
	Page    int64   `json:"page"`
	PerPage int64   `json:"perPage"`
	Users   []*User `json:"users"`
}

// SearchUsers performs a text search for users.
func (c *Client) SearchUsers(params SearchUsersParams) (*SearchUsersResponse, error) {
	q := NewQuery(c, "/search/users")
	q.AddString("query", params.Query)
	q.AddInt64IfNonZero("page", params.Page)
	r := &SearchUsersResponse{}
	return r, q.Get(r)
}
