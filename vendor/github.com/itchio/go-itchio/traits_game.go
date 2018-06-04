package itchio

import (
	"encoding/json"
	"reflect"
)

var _ json.Marshaler = GameTraits{}
var _ json.Unmarshaler = (*GameTraits)(nil)

var gameTraitMap map[string]int
var gameTraitList []string

func init() {
	typ := reflect.TypeOf(GameTraits{})
	gameTraitList = make([]string, typ.NumField())
	gameTraitMap = make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		trait := typ.Field(i).Tag.Get("trait")
		gameTraitMap[trait] = i
		gameTraitList[i] = trait
	}
}

func (tt GameTraits) MarshalJSON() ([]byte, error) {
	var traits []string
	val := reflect.ValueOf(tt)
	for i, trait := range gameTraitList {
		if val.Field(i).Bool() {
			traits = append(traits, trait)
		}
	}
	return json.Marshal(traits)
}

func (tt *GameTraits) UnmarshalJSON(data []byte) error {
	var traits []string
	err := json.Unmarshal(data, &traits)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(tt).Elem()
	for _, trait := range traits {
		val.Field(gameTraitMap[trait]).SetBool(true)
	}
	return nil
}

func GameTraitHookFunc(
	f reflect.Type,
	t reflect.Type,
	data interface{}) (interface{}, error) {

	if t != reflect.TypeOf(GameTraits{}) {
		return data, nil
	}

	var tt GameTraits
	if f.Kind() != reflect.Slice {
		// oh yeah lua will emit `{}` for empty arrays,
		// which golang unmarshals into a `map[string]interface{}`
		// so let's ignore those
		return tt, nil
	}

	val := reflect.ValueOf(&tt).Elem()

	var traits = data.([]interface{})
	for _, k := range traits {
		if trait, ok := k.(string); ok {
			val.Field(gameTraitMap[trait]).SetBool(true)
		}
	}

	return tt, nil
}
