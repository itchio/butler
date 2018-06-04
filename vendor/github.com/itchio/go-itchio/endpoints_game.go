package itchio

type GetGameParams struct {
	GameID int64

	Credentials GameCredentials
}

// GetGameResponse is what the API server responds when we ask for a game's info
type GetGameResponse struct {
	Game *Game `json:"game"`
}

func (c *Client) GetGame(p GetGameParams) (*GetGameResponse, error) {
	q := NewQuery(c, "/games/%d", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &GetGameResponse{}
	return r, q.Get(r)
}
