package manifest

import "github.com/itchio/butler/manager"

// A Manifest describes prerequisites (dependencies) and actions that
// can be taken while launching a game.
type Manifest struct {
	// Actions are a list of options to give the user when launching a game.
	Actions []*Action `json:"actions"`

	// Prereqs describe libraries or frameworks that must be installed
	// prior to launching a game
	Prereqs []*Prereq `json:"prereqs"`
}

// An Action is a choice for the user to pick when launching a game.
//
// see https://itch.io/docs/itch/integrating/manifest.html
type Action struct {
	// human-readable or standard name
	Name string `json:"name"`

	// file path (relative to manifest or absolute), URL, etc.
	Path string `json:"path"`

	// icon name (see static/fonts/icomoon/demo.html, don't include `icon-` prefix)
	Icon string `json:"icon"`

	// command-line arguments
	Args []string `json:"args"`

	// sandbox opt-in
	Sandbox bool `json:"sandbox"`

	// requested API scope
	Scope string `json:"scope"`

	// don't redirect stdout/stderr, open in new console window
	Console bool `json:"console"`

	// platform to restrict this action too
	Platform manager.ItchPlatform `json:"platform"`

	// localized action name
	Locales map[string]*ActionLocale `json:"locales"`
}

type Prereq struct {
	// A prerequisite to be installed, see <https://itch.io/docs/itch/integrating/prereqs/> for the full list.
	Name string `json:"name"`
}

type ActionLocale struct {
	// A localized action name
	Name string `json:"name"`
}
