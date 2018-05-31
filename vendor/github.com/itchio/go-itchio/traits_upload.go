package itchio

import (
	"encoding/json"
	"reflect"
)

type UploadTraits struct {
	PlatformWindows bool `trait:"p_windows"`
	PlatformLinux   bool `trait:"p_linux"`
	PlatformOSX     bool `trait:"p_osx"`
	PlatformAndroid bool `trait:"p_android"`
	Preorder        bool `trait:"preorder"`
	Demo            bool `trait:"demo"`
}

var _ json.Marshaler = UploadTraits{}
var _ json.Unmarshaler = (*UploadTraits)(nil)

var uploadTraitMap map[string]int
var uploadTraitList []string

func init() {
	typ := reflect.TypeOf(UploadTraits{})
	uploadTraitList = make([]string, typ.NumField())
	uploadTraitMap = make(map[string]int)
	for i := 0; i < typ.NumField(); i++ {
		trait := typ.Field(i).Tag.Get("trait")
		uploadTraitMap[trait] = i
		uploadTraitList[i] = trait
	}
}

func (tt UploadTraits) MarshalJSON() ([]byte, error) {
	var traits []string
	val := reflect.ValueOf(tt)
	for i, trait := range uploadTraitList {
		if val.Field(i).Bool() {
			traits = append(traits, trait)
		}
	}
	return json.Marshal(traits)
}

func (tt *UploadTraits) UnmarshalJSON(data []byte) error {
	var traits []string
	err := json.Unmarshal(data, &traits)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(tt).Elem()
	for _, trait := range traits {
		val.Field(uploadTraitMap[trait]).SetBool(true)
	}
	return nil
}

func UploadTraitHookFunc(
	f reflect.Type,
	t reflect.Type,
	data interface{}) (interface{}, error) {

	if t != reflect.TypeOf(UploadTraits{}) {
		return data, nil
	}

	var tt UploadTraits
	if f.Kind() != reflect.Slice {
		// lua emits `{}` for empty arrays, golang unmarshals that into a `map[string]interface{}`
		// they're empty anyway
		return tt, nil
	}

	val := reflect.ValueOf(&tt).Elem()

	var traits = data.([]interface{})
	for _, k := range traits {
		if trait, ok := k.(string); ok {
			val.Field(uploadTraitMap[trait]).SetBool(true)
		}
	}

	return tt, nil
}
