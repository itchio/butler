package shortcut

import "github.com/itchio/headway/state"

type CreateParams struct {
	// What the user should see
	DisplayName string

	// Path to icon file
	IconSource string

	// What the shortcut should open
	URL string

	// For logging
	Consumer *state.Consumer
}
