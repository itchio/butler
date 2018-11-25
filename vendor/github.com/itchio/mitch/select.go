package mitch

import (
	"fmt"
	"reflect"
)

func (s *Store) SelectOne(dst interface{}, src *ValuesSort, eq Eq) bool {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Type().Kind() != reflect.Ptr {
		panic("expected to selectOne into a pointer")
	}

	dstVal = dstVal.Elem()

	srcVal := reflect.ValueOf(src.Apply())
	if srcVal.Type().Kind() != reflect.Slice {
		panic(fmt.Sprintf("expected to select from a slice (but was %v)", srcVal.Type()))
	}

	for i := 0; i < srcVal.Len(); i++ {
		el := srcVal.Index(i)
		if eq.Match(el) {
			dstVal.Set(el.Elem())
			return true
		}
	}
	return false
}

func (s *Store) Select(dst interface{}, src *ValuesSort, eq Eq) {
	dstVal := reflect.ValueOf(dst)
	if dstVal.Type().Kind() != reflect.Ptr {
		panic("expected to select into a pointer to slice (not a ptr)")
	}

	dstVal = dstVal.Elem()
	if dstVal.Type().Kind() != reflect.Slice {
		panic("expected to select into a pointer to slice (ptr to not a slice)")
	}

	srcVal := reflect.ValueOf(src.Apply())
	if srcVal.Type().Kind() != reflect.Slice {
		panic(fmt.Sprintf("expected to select from a slice (but was %v)", srcVal.Type()))
	}

	for i := 0; i < srcVal.Len(); i++ {
		el := srcVal.Index(i)
		if eq.Match(el) {
			dstVal.Set(reflect.Append(dstVal, el))
		}
	}
}
