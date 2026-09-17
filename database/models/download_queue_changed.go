package models

import (
	"slices"
	"sync"
)

// DownloadQueueChanged wakes the downloads driver when something changes
// what it should be doing: a download queued, retried, discarded or moved
// to the front. hades reports every write it makes to downloads, so callers
// don't notify. Raw SQL against that table would have to call Notify
// itself, and so would a write inside a transaction, after committing.
var DownloadQueueChanged = &ChangeSignal{ch: make(chan struct{})}

type ChangeSignal struct {
	mu sync.Mutex
	ch chan struct{}
}

// Wait returns a channel closed by the next Notify. Take it before
// reading the state being waited on, so a change in between is not missed.
func (s *ChangeSignal) Wait() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ch
}

func (s *ChangeSignal) Notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	close(s.ch)
	s.ch = make(chan struct{})
}

// afterWrite is hades' AfterWrite hook.
func afterWrite(tables []string) {
	if slices.Contains(tables, hadesContext.TableName(&Download{})) {
		DownloadQueueChanged.Notify()
	}
}
