package itchio

import "time"

/////////////////////////////
// types
/////////////////////////////

// UserGameInteractionsSummary gives the latest "run at" timestamp and the
// sum of seconds run for all sessions.
type UserGameInteractionsSummary struct {
	SecondsRun int64      `json:"seconds_run"`
	LastRunAt  *time.Time `json:"last_run_at"`
}

// UserGameSession represents a single continuous run for a game.
type UserGameSession struct {
	// ID is the global itch.io identifier for this session
	ID int64 `json:"id"`
	// SecondsRun is the number of seconds the game has run during this session.
	SecondsRun int64 `json:"seconds_run"`
	// LastRunAt is the time this session ended.
	LastRunAt *time.Time `json:"last_run_at"`
}

/////////////////////////////
// endpoints
/////////////////////////////

// GetUserGameSessionsParams : params for GetUserGameSessions
type GetUserGameSessionsParams struct {
	GameID int64

	Credentials GameCredentials
}

// GetUserGameSessionsResponse : response for GetUserGameSessions
type GetUserGameSessionsResponse struct {
	Summary          UserGameInteractionsSummary `json:"summary"`
	UserGameSessions []*UserGameSession          `json:"user_game_sessions"`
}

// GetUserGameSessions retrieves a summary of interactions with a game by user,
// and the most recent sessions.
func (c *Client) GetUserGameSessions(p GetUserGameSessionsParams) (*GetUserGameSessionsResponse, error) {
	q := NewQuery(c, "/games/%d/interactions/sessions", p.GameID)
	q.AddGameCredentials(p.Credentials)
	r := &GetUserGameSessionsResponse{}
	return r, q.Get(r)
}

// CreateUserGameSessionParams : params for CreateUserGameSession
type CreateUserGameSessionParams struct {
	// ID of the game this session is for
	GameID int64
	// Time the game has run (so far), in seconds
	SecondsRun int64
	// End of the session (so far). This is not the same
	// as the request time, because the session may be "uploaded"
	// later than it is being recorded. This happens especially
	// if the session was recorded when offline.
	LastTouchedAt *time.Time
	// Upload being run this session
	UploadID int64
	// Optional (if the upload is not wharf-enabled): build being run this session
	BuildID int64

	// Download key etc., in case this is a paid game
	Credentials GameCredentials
}

// CreateUserGameSessionResponse : response for CreateUserGameSession
type CreateUserGameSessionResponse struct {
	// A summary of interactions for this user+game
	Summary *UserGameInteractionsSummary `json:"summary"`
	// The freshly-created game session
	UserGameSession *UserGameSession `json:"user_game_session"`
}

// CreateUserGameSession creates a session for a user/game. It can
// be later updated.
func (c *Client) CreateUserGameSession(p CreateUserGameSessionParams) (*CreateUserGameSessionResponse, error) {
	q := NewQuery(c, "/games/%d/interactions/sessions", p.GameID)
	q.AddGameCredentials(p.Credentials)
	q.AddInt64("seconds_run", p.SecondsRun)
	q.AddTimePtr("last_touched_at", p.LastTouchedAt)
	q.AddInt64("upload_id", p.UploadID)
	q.AddInt64IfNonZero("build_id", p.BuildID)
	r := &CreateUserGameSessionResponse{}
	return r, q.Post(r)
}

// UpdateUserGameSessionParams : params for UpdateUserGameSession
// Note that upload_id and build_id are fixed on creation, so they
// can't be updated.
type UpdateUserGameSessionParams struct {
	// The ID of the session to update. It must already exist.
	SessionID int64
	// The ID of the game this session is for
	GameID int64

	SecondsRun    int64
	LastTouchedAt *time.Time

	Credentials GameCredentials
}

// UpdateUserGameSessionResponse : response for UpdateUserGameSession
type UpdateUserGameSessionResponse struct {
	Summary         *UserGameInteractionsSummary `json:"summary"`
	UserGameSession *UserGameSession             `json:"user_game_session"`
}

// UpdateUserGameSession updates an existing user+game session with a new
// duration and timestamp.
func (c *Client) UpdateUserGameSession(p UpdateUserGameSessionParams) (*UpdateUserGameSessionResponse, error) {
	q := NewQuery(c, "/games/%d/interactions/sessions/%d", p.GameID, p.SessionID)
	q.AddGameCredentials(p.Credentials)
	q.AddInt64("seconds_run", p.SecondsRun)
	q.AddTimePtr("last_touched_at", p.LastTouchedAt)
	r := &UpdateUserGameSessionResponse{}
	return r, q.Post(r)
}
