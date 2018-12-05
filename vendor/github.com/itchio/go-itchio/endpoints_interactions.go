package itchio

import "time"

/////////////////////////////
// types
/////////////////////////////

// UserGameInteractionsSummary gives the latest "run at" timestamp and the
// sum of seconds run for all sessions.
type UserGameInteractionsSummary struct {
	SecondsRun int64      `json:"secondsRun"`
	LastRunAt  *time.Time `json:"lastRunAt"`
}

// UserGameSession represents a single continuous run for a game.
type UserGameSession struct {
	// ID is the global itch.io identifier for this session
	ID int64 `json:"id"`
	// SecondsRun is the number of seconds the game has run during this session.
	SecondsRun int64 `json:"secondsRun"`
	// LastRunAt is the time this session ended.
	LastRunAt *time.Time `json:"lastRunAt"`
}

/////////////////////////////
// endpoints
/////////////////////////////

// GetUserGameSessionsParams : params for GetUserGameSessions
type GetUserGameSessionsParams struct {
	GameID int64

	Credentials GameCredentials
}

type SessionPlatform string

const (
	SessionPlatformLinux   SessionPlatform = "linux"
	SessionPlatformMacOS   SessionPlatform = "macos"
	SessionPlatformWindows SessionPlatform = "windows"
)

type SessionArchitecture string

const (
	SessionArchitecture386   SessionArchitecture = "386"
	SessionArchitectureAmd64 SessionArchitecture = "amd64"
)

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
	LastRunAt *time.Time
	// Upload being run this session
	UploadID int64
	// Optional (if the upload is not wharf-enabled): build being run this session
	BuildID int64

	Platform     SessionPlatform
	Architecture SessionArchitecture

	// Download key etc., in case this is a paid game
	Credentials GameCredentials
}

// CreateUserGameSessionResponse : response for CreateUserGameSession
type CreateUserGameSessionResponse struct {
	// A summary of interactions for this user+game
	Summary *UserGameInteractionsSummary `json:"summary"`
	// The freshly-created game session
	UserGameSession *UserGameSession `json:"userGameSession"`
}

// CreateUserGameSession creates a session for a user/game. It can
// be later updated.
func (c *Client) CreateUserGameSession(p CreateUserGameSessionParams) (*CreateUserGameSessionResponse, error) {
	q := NewQuery(c, "/profile/game-sessions")
	q.AddGameCredentials(p.Credentials)
	q.AddInt64("game_id", p.GameID)
	q.AddInt64("seconds_run", p.SecondsRun)
	q.AddTimePtr("last_run_at", p.LastRunAt)
	q.AddInt64("upload_id", p.UploadID)
	q.AddInt64IfNonZero("build_id", p.BuildID)
	q.AddStringIfNonEmpty("platform", string(p.Platform))
	q.AddStringIfNonEmpty("architecture", string(p.Architecture))
	r := &CreateUserGameSessionResponse{}
	return r, q.Post(r)
}

// UpdateUserGameSessionParams : params for UpdateUserGameSession
// Note that upload_id and build_id are fixed on creation, so they
// can't be updated.
type UpdateUserGameSessionParams struct {
	// The ID of the session to update. It must already exist.
	SessionID int64

	SecondsRun int64
	LastRunAt  *time.Time
	Crashed    bool
}

// UpdateUserGameSessionResponse : response for UpdateUserGameSession
type UpdateUserGameSessionResponse struct {
	Summary         *UserGameInteractionsSummary `json:"summary"`
	UserGameSession *UserGameSession             `json:"userGameSession"`
}

// UpdateUserGameSession updates an existing user+game session with a new
// duration and timestamp.
func (c *Client) UpdateUserGameSession(p UpdateUserGameSessionParams) (*UpdateUserGameSessionResponse, error) {
	q := NewQuery(c, "/profile/game-sessions/%d", p.SessionID)
	q.AddInt64IfNonZero("seconds_run", p.SecondsRun)
	q.AddTimePtr("last_run_at", p.LastRunAt)
	q.AddBoolIfTrue("crashed", true)
	r := &UpdateUserGameSessionResponse{}
	return r, q.Post(r)
}

type GetGameSessionsSummaryResponse struct {
	Summary *UserGameInteractionsSummary `json:"summary"`
}

// GetGameSessionsSummary returns a summary of game sessions for a given game.
func (c *Client) GetGameSessionsSummary(gameID int64) (*GetGameSessionsSummaryResponse, error) {
	q := NewQuery(c, "/profile/game-sessions/summaries/%d", gameID)
	r := &GetGameSessionsSummaryResponse{}
	return r, q.Post(r)
}
