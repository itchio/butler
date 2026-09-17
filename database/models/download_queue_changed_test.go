package models

import (
	"testing"
	"time"
)

func TestChangeSignal(t *testing.T) {
	s := &ChangeSignal{ch: make(chan struct{})}

	first, second := s.Wait(), s.Wait()
	select {
	case <-first:
		t.Fatal("closed before any notify")
	default:
	}

	s.Notify()
	for _, ch := range []<-chan struct{}{first, second} {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("waiter not woken by notify")
		}
	}

	select {
	case <-s.Wait():
		t.Fatal("a wait taken after notify must need another one")
	default:
	}
}
