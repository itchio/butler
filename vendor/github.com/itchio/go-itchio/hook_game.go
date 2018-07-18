package itchio

import (
	"reflect"
)

func GameHookFunc(
	f reflect.Type,
	t reflect.Type,
	data interface{}) (interface{}, error) {

	if t != reflect.TypeOf(Game{}) {
		return data, nil
	}

	if gameMap, ok := data.(map[string]interface{}); ok {
		// API v2 briefly returned traits, which were 100%
		// amos's terrible idea - they're going away, but
		// in the meantime let's convert them to something saner.
		if traitsAny, ok := gameMap["traits"]; ok {
			platforms := make(map[string]interface{})
			if traits, ok := traitsAny.([]interface{}); ok {
				for _, traitAny := range traits {
					if trait, ok := traitAny.(string); ok {
						switch trait {
						case "p_osx":
							platforms["osx"] = ArchitecturesAll
						case "p_windows":
							platforms["windows"] = ArchitecturesAll
						case "p_linux":
							platforms["linux"] = ArchitecturesAll
						case "can_be_bought":
							gameMap["canBeBought"] = true
						case "has_demo":
							gameMap["hasDemo"] = true
						case "in_press_system":
							gameMap["inPressSystem"] = true
						}
					}
				}
			}
			gameMap["platforms"] = platforms
			delete(gameMap, "traits")
			return gameMap, nil
		}
	}

	return data, nil
}
