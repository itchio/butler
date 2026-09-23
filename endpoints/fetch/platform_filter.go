package fetch

import (
	"strings"

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

// condForScannedPlatformsFilter returns a condition matching games whose
// scanned_platforms JSON array contains any of the given entries, or nil
// for an empty list. Unscanned games (NULL column) never match. The query
// must join the games table when the condition is non-nil.
func condForScannedPlatformsFilter(platforms []string) builder.Cond {
	if len(platforms) == 0 {
		return nil
	}
	args := make([]any, len(platforms))
	marks := make([]string, len(platforms))
	for i, p := range platforms {
		args[i] = p
		marks[i] = "?"
	}
	return builder.Expr(
		"exists (select 1 from json_each(games.scanned_platforms) where json_each.value in ("+strings.Join(marks, ",")+"))",
		args...,
	)
}
