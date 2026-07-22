package launch

import (
	"context"
	"time"

	"github.com/itchio/butler/butlerd/horror"
	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

type sessionClient interface {
	CreateUserGameSession(ctx context.Context, p itchio.CreateUserGameSessionParams) (*itchio.CreateUserGameSessionResponse, error)
	UpdateUserGameSession(ctx context.Context, p itchio.UpdateUserGameSessionParams) (*itchio.UpdateUserGameSessionResponse, error)
}

const defaultSessionUpdateInterval = 1 * time.Minute

// Leave enough time for the API client's retry backoff (1+2+4+8+16s).
const finalSessionUpdateTimeout = 35 * time.Second

const sessionWatcherJoinGrace = 5 * time.Second

type sessionTracker struct {
	consumer *state.Consumer
	client   sessionClient

	gameID       int64
	uploadID     int64
	buildID      int64
	credentials  itchio.GameCredentials
	platform     itchio.SessionPlatform
	architecture itchio.SessionArchitecture

	persistSummary func(summary *itchio.UserGameInteractionsSummary)

	updateInterval     time.Duration
	finalUpdateTimeout time.Duration
}

func (st *sessionTracker) interval() time.Duration {
	if st.updateInterval > 0 {
		return st.updateInterval
	}
	return defaultSessionUpdateInterval
}

func (st *sessionTracker) finalTimeout() time.Duration {
	if st.finalUpdateTimeout > 0 {
		return st.finalUpdateTimeout
	}
	return finalSessionUpdateTimeout
}

// Keep at's monotonic reading so network delays and wall-clock adjustments do
// not affect the recorded playtime.
type sessionEnd struct {
	at      time.Time
	crashed bool
}

func (st *sessionTracker) run(ctx context.Context, started <-chan time.Time, ended <-chan sessionEnd) {
	var startedAt time.Time
	var end sessionEnd
	gameRunning := false

	select {
	case <-ctx.Done():
		st.consumer.Debugf("Launch cancelled before session started, nothing to record")
		return
	case end = <-ended:
		// Both channels may be ready for a short session, and select may pick ended first.
		select {
		case startedAt = <-started:
		default:
			st.consumer.Debugf("Launch ended without starting a tracked session, nothing to record")
			return
		}
	case startedAt = <-started:
		gameRunning = true
	}

	sessionID, err := st.createSession(ctx, startedAt)
	if err != nil {
		st.consumer.Warnf("Initial session creation: %+v", err)
		return
	}

	if gameRunning {
		ticker := time.NewTicker(st.interval())
		defer ticker.Stop()

	regularUpdates:
		for {
			select {
			case <-ctx.Done():
				st.consumer.Debugf("Launch cancelled while running, skipping final update")
				return
			case <-ticker.C:
				if err := st.updateSession(ctx, sessionID, startedAt, time.Now(), false); err != nil {
					st.consumer.Warnf("Regular session update: %+v", err)
				}
			case end = <-ended:
				st.consumer.Debugf("Session ended!")
				break regularUpdates
			}
		}
	}

	// Request teardown should not cancel the final update.
	finalCtx, cancel := context.WithTimeout(context.Background(), st.finalTimeout())
	defer cancel()
	if err := st.updateSession(finalCtx, sessionID, startedAt, end.at, end.crashed); err != nil {
		st.consumer.Warnf("Final session update: %+v", err)
		return
	}

	st.consumer.Debugf("Entire session committed successfully!")
}

func (st *sessionTracker) createSession(ctx context.Context, startedAt time.Time) (sessionID int64, retErr error) {
	defer horror.RecoverInto(&retErr)

	lastRunAt := startedAt.UTC()
	res, err := st.client.CreateUserGameSession(ctx, itchio.CreateUserGameSessionParams{
		GameID:       st.gameID,
		UploadID:     st.uploadID,
		BuildID:      st.buildID,
		Credentials:  st.credentials,
		Platform:     st.platform,
		Architecture: st.architecture,

		SecondsRun: 0,
		LastRunAt:  &lastRunAt,
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}
	if res.UserGameSession == nil {
		return 0, errors.New("session creation returned no session")
	}

	st.persist(res.Summary)
	return res.UserGameSession.ID, nil
}

func (st *sessionTracker) updateSession(ctx context.Context, sessionID int64, startedAt, at time.Time, crashed bool) (retErr error) {
	defer horror.RecoverInto(&retErr)

	secondsRun := int64(at.Sub(startedAt).Seconds())
	lastRunAt := at.UTC()

	res, err := st.client.UpdateUserGameSession(ctx, itchio.UpdateUserGameSessionParams{
		SessionID: sessionID,

		SecondsRun: secondsRun,
		LastRunAt:  &lastRunAt,
		Crashed:    crashed,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	st.persist(res.Summary)
	return nil
}

func (st *sessionTracker) persist(summary *itchio.UserGameInteractionsSummary) {
	if summary == nil || st.persistSummary == nil {
		return
	}
	st.persistSummary(summary)
}
