package itchio

type SearchGamesParams struct {
	Query string
	Page  int64
}

type SearchGamesResponse struct {
	Page    int64   `json:"page"`
	PerPage int64   `json:"perPage"`
	Games   []*Game `json:"games"`
}

func (c *Client) SearchGames(params SearchGamesParams) (*SearchGamesResponse, error) {
	q := NewQuery(c, "/search/games")
	q.AddString("query", params.Query)
	q.AddInt64IfNonZero("page", params.Page)
	r := &SearchGamesResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type SearchUsersParams struct {
	Query string
	Page  int64
}

type SearchUsersResponse struct {
	Page    int64   `json:"page"`
	PerPage int64   `json:"perPage"`
	Users   []*User `json:"users"`
}

func (c *Client) SearchUsers(params SearchUsersParams) (*SearchUsersResponse, error) {
	q := NewQuery(c, "/search/users")
	q.AddString("query", params.Query)
	q.AddInt64IfNonZero("page", params.Page)
	r := &SearchUsersResponse{}
	return r, q.Get(r)
}
