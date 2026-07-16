package launch

import (
	"testing"

	"github.com/itchio/butler/butlerd"
	"github.com/itchio/butler/database/models"
	"github.com/itchio/headway/state"
	"github.com/itchio/hush/manifest"
)

func makeTarget(name string, path string) *butlerd.LaunchTarget {
	return &butlerd.LaunchTarget{
		Action: &manifest.Action{
			Name: name,
			Path: path,
		},
		Strategy: &butlerd.StrategyResult{},
	}
}

func TestFindTarget(t *testing.T) {
	t.Parallel()

	play := makeTarget("play", "./game.exe")
	editor := makeTarget("editor", "editor.exe")
	html := makeTarget("index.html (13.0 KiB)", "index.html")
	targets := []*butlerd.LaunchTarget{play, editor, html}

	if got := findTarget(targets, "play"); got != play {
		t.Errorf("expected match by action name, got %v", got)
	}

	if got := findTarget(targets, "game.exe"); got != play {
		t.Errorf("expected match by path despite ./ prefix, got %v", got)
	}

	if got := findTarget(targets, "./editor.exe"); got != editor {
		t.Errorf("expected match by ./-prefixed path, got %v", got)
	}

	if got := findTarget(targets, "index.html"); got != html {
		t.Errorf("expected match by candidate path, got %v", got)
	}

	if got := findTarget(targets, "soundtrack.mp3"); got != nil {
		t.Errorf("expected no match, got %v", got)
	}

	// a name match wins over a path match
	decoy := makeTarget("game.exe", "other.exe")
	if got := findTarget([]*butlerd.LaunchTarget{play, decoy}, "game.exe"); got != decoy {
		t.Errorf("expected name match to win over path match, got %v", got)
	}
}

func TestSettingsLaunchTarget(t *testing.T) {
	t.Parallel()

	rc := &butlerd.RequestContext{Consumer: &state.Consumer{}}

	cave := &models.Cave{
		Settings: models.JSON(`{"launchTarget":"index.html"}`),
	}
	if got := settingsLaunchTarget(rc, cave); got != "index.html" {
		t.Errorf("expected cave settings launch target, got %q", got)
	}

	if got := settingsLaunchTarget(rc, &models.Cave{}); got != "" {
		t.Errorf("expected no preference for empty settings, got %q", got)
	}

	broken := &models.Cave{Settings: models.JSON(`{nope`)}
	if got := settingsLaunchTarget(rc, broken); got != "" {
		t.Errorf("expected no preference for broken settings, got %q", got)
	}
}
