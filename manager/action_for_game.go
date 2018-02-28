package manager

import itchio "github.com/itchio/go-itchio"

type ClassificationAction string

const (
	ClassificationActionOpen   ClassificationAction = "open"
	ClassificationActionLaunch                      = "launch"
)

var knownClassificationActions = map[string]ClassificationAction{
	"game": ClassificationActionLaunch,
	"tool": ClassificationActionLaunch,

	"assets":        ClassificationActionOpen,
	"game_mod":      ClassificationActionOpen,
	"physical_game": ClassificationActionOpen,
	"soundtrack":    ClassificationActionOpen,
	"other":         ClassificationActionOpen,
	"comic":         ClassificationActionOpen,
	"book":          ClassificationActionOpen,
}

func actionForGame(game *itchio.Game) ClassificationAction {
	knownAction, ok := knownClassificationActions[string(game.Classification)]
	if ok {
		return knownAction
	}

	return ClassificationActionLaunch
}
