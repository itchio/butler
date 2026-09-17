package models

import (
	"testing"
	"time"

	"xorm.io/builder"
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

func woken(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestDownloadWritesNotify(t *testing.T) {
	conn := bundleTestConn(t)

	changed := DownloadQueueChanged.Wait()
	MustSave(conn, &Cave{ID: "cave"})
	if woken(changed) {
		t.Fatal("saving another model woke the driver")
	}

	MustSave(conn, &Download{ID: "dl", CaveID: "cave"})
	if !woken(changed) {
		t.Fatal("saving a download did not wake the driver")
	}

	changed = DownloadQueueChanged.Wait()
	DiscardDownloadsByCaveID(conn, "cave")
	if !woken(changed) {
		t.Fatal("a bulk update of downloads did not wake the driver")
	}

	changed = DownloadQueueChanged.Wait()
	MustDelete(conn, &Download{}, builder.Eq{"id": "dl"})
	if !woken(changed) {
		t.Fatal("deleting downloads did not wake the driver")
	}
}
