package itchio

// GetGameParams : params for GetGame
type GetGameParams struct {
	GameID int64

	Credentials GameCredentials
}

// GetGameResponse : response for GetGame
type GetGameResponse struct {
	Game *Game `json:"game"`
}

// GetGame retrieves a single game by ID.
func (c *Client) GetGame(p GetGameParams) (*GetGameResponse, error) {
	q := NewQuery(c, "/games/%d", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &GetGameResponse{}
	return r, q.Get(r)
}
