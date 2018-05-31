package hades

import (
	"reflect"
	"time"
)

func DBValue(x interface{}) interface{} {
	typ := reflect.TypeOf(x)
	value := reflect.ValueOf(x)
	wasPtr := false

	if typ.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}

		wasPtr = true
		typ = typ.Elem()
		value = value.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		if value.Bool() {
			return 1
		}
		return 0
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return value.Interface().(time.Time).Format(time.RFC3339Nano)
		}
	}

	if wasPtr {
		return value.Interface()
	}
	return x
}
