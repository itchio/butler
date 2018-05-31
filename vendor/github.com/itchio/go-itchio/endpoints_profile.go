package itchio

//-------------------------------------------------------

// GetProfileResponse is what the API server responds when we ask for the user's profile
type GetProfileResponse struct {
	User *User `json:"user"`
}

// GetProfile returns information about the user the current credentials belong to
func (c *Client) GetProfile() (*GetProfileResponse, error) {
	q := NewQuery(c, "/profile")
	r := &GetProfileResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// ListProfileGamesResponse is what the API server answers when we ask for what games
// an account develops.
type ListProfileGamesResponse struct {
	Games []*Game `json:"games"`
}

// ListProfileGames lists the games one develops (ie. can edit)
func (c *Client) ListProfileGames() (*ListProfileGamesResponse, error) {
	q := NewQuery(c, "/profile/games")
	r := &ListProfileGamesResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// ListProfileOwnedKeysResponse is the response for /profile/owned-keys
type ListProfileOwnedKeysResponse struct {
	OwnedKeys []*DownloadKey `json:"ownedKeys"`
}

// ListProfileOwnedKeys lists the download keys one owns
func (c *Client) ListProfileOwnedKeys() (*ListProfileOwnedKeysResponse, error) {
	q := NewQuery(c, "/profile/owned-keys")
	r := &ListProfileOwnedKeysResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

// ListProfileCollectionsResponse is the response for /profile/collections
type ListProfileCollectionsResponse struct {
	Collections []*Collection `json:"collections"`
}

// ListProfileCollections lists the collections associated to a profile
func (c *Client) ListProfileCollections() (*ListProfileCollectionsResponse, error) {
	q := NewQuery(c, "/profile/collections")
	r := &ListProfileCollectionsResponse{}
	return r, q.Get(r)
}
