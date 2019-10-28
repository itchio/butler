package installer

import (
	"fmt"
	"time"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/installer/bfs"
)

type InstallEventSink struct {
	Append func(ev butlerd.InstallEvent) error
}

func (ies *InstallEventSink) PostEvent(event butlerd.InstallEvent) error {
	if ies == nil {
		return nil
	}

	event.Timestamp = time.Now()
	switch true {
	case event.Install != nil:
		event.Type = butlerd.InstallEventInstall
	case event.Heal != nil:
		event.Type = butlerd.InstallEventHeal
	case event.Upgrade != nil:
		event.Type = butlerd.InstallEventUpgrade
	case event.Problem != nil:
		event.Type = butlerd.InstallEventProblem
	case event.GhostBusting != nil:
		event.Type = butlerd.InstallEventGhostBusting
	case event.Patching != nil:
		event.Type = butlerd.InstallEventPatching
	case event.Fallback != nil:
		event.Type = butlerd.InstallEventFallback
	}

	if event.Type == "" {
		// wee runtime checks
		panic("InstallEventSink events shoudl always have Type set")
	}

	return ies.Append(event)
}

func (ies *InstallEventSink) PostProblem(err error) error {
	prob := ies.MakeProblem(err)
	return ies.PostEvent(butlerd.InstallEvent{
		Type:    butlerd.InstallEventProblem,
		Problem: &prob,
	})
}

func (ies *InstallEventSink) MakeProblem(err error) butlerd.ProblemInstallEvent {
	return butlerd.ProblemInstallEvent{
		Error:      fmt.Sprintf("%v", err),
		ErrorStack: fmt.Sprintf("%+v", err),
	}
}

func (ies *InstallEventSink) PostGhostBusting(operation string, stats bfs.BustGhostStats) error {
	return ies.PostEvent(butlerd.InstallEvent{
		GhostBusting: &butlerd.GhostBustingInstallEvent{
			Operation: "heal",
			Found:     stats.Found,
			Removed:   stats.Removed,
		},
	})
}
