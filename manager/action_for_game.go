package manager

import itchio "github.com/itchio/go-itchio"

type ClassificationAction string

const (
	ClassificationActionOpen   ClassificationAction = "open"
	ClassificationActionLaunch ClassificationAction = "launch"
)

var knownClassificationActions = map[itchio.GameClassification]ClassificationAction{
	itchio.GameClassificationGame: ClassificationActionLaunch,
	itchio.GameClassificationTool: ClassificationActionLaunch,

	itchio.GameClassificationAssets:       ClassificationActionOpen,
	itchio.GameClassificationGameMod:      ClassificationActionOpen,
	itchio.GameClassificationPhysicalGame: ClassificationActionOpen,
	itchio.GameClassificationSoundtrack:   ClassificationActionOpen,
	itchio.GameClassificationOther:        ClassificationActionOpen,
	itchio.GameClassificationComic:        ClassificationActionOpen,
	itchio.GameClassificationBook:         ClassificationActionOpen,
}

func actionForGame(game *itchio.Game) ClassificationAction {
	knownAction, ok := knownClassificationActions[game.Classification]
	if ok {
		return knownAction
	}

	return ClassificationActionLaunch
}
