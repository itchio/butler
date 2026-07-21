package launch

import (
	"context"
	"sync"
	"testing"
	"time"

	itchio "github.com/itchio/go-itchio"
	"github.com/itchio/headway/state"
	"github.com/pkg/errors"
)

type fakeSessionClient struct {
	mu sync.Mutex

	createCalls int
	updates     []itchio.UpdateUserGameSessionParams

	createErr   error
	updateDelay time.Duration
	updated     chan struct{}

	nextID int64
}

func (f *fakeSessionClient) CreateUserGameSession(ctx context.Context, p itchio.CreateUserGameSessionParams) (*itchio.CreateUserGameSessionResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.nextID++
	return &itchio.CreateUserGameSessionResponse{
		Summary:         &itchio.UserGameInteractionsSummary{},
		UserGameSession: &itchio.UserGameSession{ID: f.nextID},
	}, nil
}

func (f *fakeSessionClient) UpdateUserGameSession(ctx context.Context, p itchio.UpdateUserGameSessionParams) (*itchio.UpdateUserGameSessionResponse, error) {
	if d := f.readUpdateDelay(); d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	f.mu.Lock()
	f.updates = append(f.updates, p)
	updated := f.updated
	f.mu.Unlock()

	if updated != nil {
		select {
		case updated <- struct{}{}:
		default:
		}
	}
	return &itchio.UpdateUserGameSessionResponse{
		Summary:         &itchio.UserGameInteractionsSummary{},
		UserGameSession: &itchio.UserGameSession{ID: p.SessionID},
	}, nil
}

func (f *fakeSessionClient) readUpdateDelay() time.Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.updateDelay
}

func (f *fakeSessionClient) snapshot() (creates int, updates []itchio.UpdateUserGameSessionParams) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls, append([]itchio.UpdateUserGameSessionParams(nil), f.updates...)
}

func newTestTracker(client sessionClient) *sessionTracker {
	return &sessionTracker{
		consumer:           &state.Consumer{},
		client:             client,
		gameID:             1,
		uploadID:           2,
		updateInterval:     time.Hour,
		finalUpdateTimeout: 2 * time.Second,
	}
}

func runTracker(st *sessionTracker, ctx context.Context, started <-chan time.Time, ended <-chan sessionEnd) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		st.run(ctx, started, ended)
	}()
	return done
}

func newChans() (chan time.Time, chan sessionEnd) {
	return make(chan time.Time, 1), make(chan sessionEnd, 1)
}

func waitDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("tracker did not return in time")
	}
}

func TestSessionCleanExit(t *testing.T) {
	client := &fakeSessionClient{}
	st := newTestTracker(client)

	started, ended := newChans()
	done := runTracker(st, context.Background(), started, ended)

	started <- time.Now()
	ended <- sessionEnd{at: time.Now(), crashed: false}
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 1 {
		t.Fatalf("expected 1 create, got %d", creates)
	}
	if len(updates) != 1 {
		t.Fatalf("expected exactly 1 (final) update, got %d", len(updates))
	}
	if updates[0].Crashed {
		t.Fatalf("final update should report crashed=false")
	}
}

func TestSessionCrashExit(t *testing.T) {
	client := &fakeSessionClient{}
	st := newTestTracker(client)

	started, ended := newChans()
	done := runTracker(st, context.Background(), started, ended)

	started <- time.Now()
	ended <- sessionEnd{at: time.Now(), crashed: true}
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 1 {
		t.Fatalf("expected 1 create, got %d", creates)
	}
	if len(updates) != 1 {
		t.Fatalf("expected exactly 1 (final) update, got %d", len(updates))
	}
	if !updates[0].Crashed {
		t.Fatalf("final update should report crashed=true")
	}
}

func TestSessionDurationIsExitBased(t *testing.T) {
	client := &fakeSessionClient{updateDelay: 300 * time.Millisecond}
	st := newTestTracker(client)

	started, ended := newChans()
	done := runTracker(st, context.Background(), started, ended)

	start := time.Now()
	started <- start
	ended <- sessionEnd{at: start.Add(5 * time.Second), crashed: false}
	waitDone(t, done)

	_, updates := client.snapshot()
	if len(updates) != 1 {
		t.Fatalf("expected 1 final update, got %d", len(updates))
	}
	if updates[0].SecondsRun != 5 {
		t.Fatalf("expected SecondsRun=5 (exit-based), got %d", updates[0].SecondsRun)
	}
}

func TestSessionEndedWithoutStart(t *testing.T) {
	client := &fakeSessionClient{}
	st := newTestTracker(client)

	started, ended := newChans() // started never sent
	done := runTracker(st, context.Background(), started, ended)

	ended <- sessionEnd{at: time.Now(), crashed: false}
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 0 || len(updates) != 0 {
		t.Fatalf("expected no session activity, got %d creates and %d updates", creates, len(updates))
	}
}

func TestSessionQuickRun(t *testing.T) {
	// Exercise both select outcomes with start and end already available.
	for i := 0; i < 200; i++ {
		client := &fakeSessionClient{}
		st := newTestTracker(client)

		started, ended := newChans()
		started <- time.Now()
		ended <- sessionEnd{at: time.Now(), crashed: false}

		done := runTracker(st, context.Background(), started, ended)
		waitDone(t, done)

		creates, updates := client.snapshot()
		if creates != 1 {
			t.Fatalf("iter %d: expected 1 create, got %d", i, creates)
		}
		if len(updates) != 1 {
			t.Fatalf("iter %d: expected 1 final update, got %d", i, len(updates))
		}
	}
}

func TestSessionSlowFinalUpdate(t *testing.T) {
	client := &fakeSessionClient{updateDelay: 200 * time.Millisecond}
	st := newTestTracker(client)

	started, ended := newChans()
	done := runTracker(st, context.Background(), started, ended)

	started <- time.Now()
	ended <- sessionEnd{at: time.Now(), crashed: false}
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 1 || len(updates) != 1 {
		t.Fatalf("expected 1 create and 1 update, got %d and %d", creates, len(updates))
	}
}

func TestSessionCancelledBeforeStart(t *testing.T) {
	client := &fakeSessionClient{}
	st := newTestTracker(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started, ended := newChans()
	done := runTracker(st, ctx, started, ended)
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 0 || len(updates) != 0 {
		t.Fatalf("expected no session activity, got %d creates and %d updates", creates, len(updates))
	}
}

func TestSessionCreateFailure(t *testing.T) {
	client := &fakeSessionClient{createErr: errors.New("boom")}
	st := newTestTracker(client)

	started, ended := newChans()
	done := runTracker(st, context.Background(), started, ended)

	started <- time.Now()
	ended <- sessionEnd{at: time.Now(), crashed: false}
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 1 {
		t.Fatalf("expected 1 create attempt, got %d", creates)
	}
	if len(updates) != 0 {
		t.Fatalf("expected no updates after create failure, got %d", len(updates))
	}
}

func TestSessionPeriodicUpdate(t *testing.T) {
	client := &fakeSessionClient{updated: make(chan struct{}, 8)}
	st := newTestTracker(client)
	st.updateInterval = 5 * time.Millisecond

	started, ended := newChans()
	done := runTracker(st, context.Background(), started, ended)

	started <- time.Now()

	select {
	case <-client.updated:
	case <-time.After(5 * time.Second):
		t.Fatal("no periodic update observed")
	}

	ended <- sessionEnd{at: time.Now(), crashed: false}
	waitDone(t, done)

	creates, updates := client.snapshot()
	if creates != 1 {
		t.Fatalf("expected 1 create, got %d", creates)
	}
	if len(updates) < 2 {
		t.Fatalf("expected at least one periodic update plus a final one, got %d", len(updates))
	}
	if updates[len(updates)-1].Crashed {
		t.Fatalf("final update should report crashed=false")
	}
}
