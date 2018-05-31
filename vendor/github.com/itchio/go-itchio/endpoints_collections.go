package itchio

//-------------------------------------------------------

type GetCollectionParams struct {
	CollectionID int64 `json:"collectionId"`
}

// GetCollectionResponse is what the API server responds when we ask for a collection's info
type GetCollectionResponse struct {
	Collection *Collection `json:"collection"`
}

func (c *Client) GetCollection(params *GetCollectionParams) (*GetCollectionResponse, error) {
	q := NewQuery(c, "/collections/%d")
	r := &GetCollectionResponse{}
	return r, q.Get(r)
}

//-------------------------------------------------------

type GetCollectionGamesParams struct {
	CollectionID int64
	Page         int64
}

type GetCollectionGamesResponse struct {
	Page            int64             `json:"page"`
	PerPage         int64             `json:"perPage"`
	CollectionGames []*CollectionGame `json:"collection_games"`
}

func (c *Client) GetCollectionGames(params *GetCollectionGamesParams) (*GetCollectionGamesResponse, error) {
	q := NewQuery(c, "/collections/%d/collection-games", params.CollectionID)
	q.AddInt64IfNonZero("page", params.Page)
	r := &GetCollectionGamesResponse{}
	return r, q.Get(r)
}
