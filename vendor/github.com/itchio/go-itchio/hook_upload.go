package itchio

import (
	"reflect"
)

// UploadHookFunc is used to transform API results from
// what they currently are to what we expect them to be.
func UploadHookFunc(
	f reflect.Type,
	t reflect.Type,
	data interface{}) (interface{}, error) {

	if t != reflect.TypeOf(Upload{}) {
		return data, nil
	}

	if uploadMap, ok := data.(map[string]interface{}); ok {
		// API v2 briefly returned traits, which were 100%
		// amos's terrible idea - they're going away, but
		// in the meantime let's convert them to something saner.
		if traitsAny, ok := uploadMap["traits"]; ok {
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
						case "demo":
							uploadMap["demo"] = true
						case "preorder":
							uploadMap["preorder"] = true
						}
					}
				}
			}
			uploadMap["platforms"] = platforms
			delete(uploadMap, "traits")
			return uploadMap, nil
		}
	}

	return data, nil
}
