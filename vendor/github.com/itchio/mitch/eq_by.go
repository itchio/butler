package mitch

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type Eq map[string]interface{}

func (eq Eq) Match(el reflect.Value) bool {
	for k, v := range eq {
		vv := el.Elem().FieldByName(k)
		if !valueEqual(reflect.ValueOf(v), vv) {
			return false
		}
	}
	return true
}

func valueEqual(a reflect.Value, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() == b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return a.Uint() == b.Uint()
	case reflect.Bool:
		return a.Bool() == b.Bool()
	case reflect.String:
		return a.String() == b.String()
	default:
		panic(fmt.Sprintf("dunno how to valueEqual kind %v", a.Kind()))
	}
}

func valueLessThan(a reflect.Value, b reflect.Value) bool {
	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a.Int() < b.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return a.Uint() < b.Uint()
	case reflect.Bool:
		if !a.Bool() {
			return true
		}
		return false
	case reflect.String:
		return strings.Compare(a.String(), b.String()) < 0
	default:
		panic(fmt.Sprintf("dunno how to valueEqual kind %v", a.Kind()))
	}
}

type valueSort struct {
	fieldName string
	dir       string
}

type ValuesSort struct {
	m     interface{}
	sorts []valueSort
}

type ValuesSortBuilder struct {
	sorts []valueSort
}

func SortBy(fieldName string, dir string) *ValuesSortBuilder {
	vsb := &ValuesSortBuilder{}
	return vsb.ThenBy(fieldName, dir)
}

func NoSort() *ValuesSortBuilder {
	return &ValuesSortBuilder{}
}

func (vsb *ValuesSortBuilder) ThenBy(fieldName string, dir string) *ValuesSortBuilder {
	if dir != "desc" && dir != "asc" {
		panic("dir must be desc or asc")
	}

	vsb.sorts = append(vsb.sorts, valueSort{
		fieldName: fieldName,
		dir:       dir,
	})
	return vsb
}

func (vsb *ValuesSortBuilder) ForMap(m interface{}) *ValuesSort {
	mapVal := reflect.ValueOf(m)
	if mapVal.Type().Kind() != reflect.Map {
		panic("Values needs a map")
	}

	return &ValuesSort{m: m, sorts: vsb.sorts}
}

func (vs *ValuesSort) Apply() interface{} {
	mapVal := reflect.ValueOf(vs.m)

	elType := mapVal.Type().Elem()
	sliceType := reflect.SliceOf(elType)
	resVal := reflect.New(sliceType).Elem()

	for _, k := range mapVal.MapKeys() {
		resVal.Set(reflect.Append(resVal, mapVal.MapIndex(k)))
	}

	for _, s := range vs.sorts {
		less := func(i int, j int) bool {
			iEl := resVal.Index(i).Elem().FieldByName(s.fieldName)
			jEl := resVal.Index(j).Elem().FieldByName(s.fieldName)
			lt := valueLessThan(iEl, jEl)
			if s.dir == "desc" {
				lt = !lt
			}
			return lt
		}
		sort.Slice(resVal.Interface(), less)
	}

	return resVal.Interface()
}
