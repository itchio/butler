package models

import "sync"

// DownloadQueueChanged wakes the downloads driver when something changes
// what it should be doing: a download queued, retried, discarded or moved
// to the front. Whatever writes such a change calls Notify once it is
// saved. It lives here so that code outside the downloads endpoints, such
// as uninstalling, can reach it.
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
