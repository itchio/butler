package itchio

type GetUserParams struct {
	UserID int64
}

// GetUserResponse is what the API server responds when we ask for a user's info
type GetUserResponse struct {
	User *User `json:"user"`
}

func (c *Client) GetUser(p GetUserParams) (*GetUserResponse, error) {
	q := NewQuery(c, "/users/%d", p.UserID)
	r := &GetUserResponse{}
	return r, q.Get(r)
}
