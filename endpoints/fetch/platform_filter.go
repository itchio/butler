package fetch

import (
	"xorm.io/builder"
)

// condForPlatformFilter returns a condition matching games that have a
// download tagged for the given platform ("windows", "linux", "osx"), or
// web-playable games ("web"). Returns nil when platform is empty. The
// query must join the games table when the condition is non-nil.
func condForPlatformFilter(platform string) builder.Cond {
	switch platform {
	case "windows", "linux", "osx":
		// Platform columns hold architecture strings and are empty (or NULL
		// on old rows) when the game has no download tagged for that platform.
		return builder.Neq{"games." + platform: ""}
	case "web":
		return builder.Eq{"games.type": "html"}
	}
	return nil
}
