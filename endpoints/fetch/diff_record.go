package fetch

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/go-errors/errors"
)

type ChangedFields map[string]interface{}

func DiffRecord(x, y interface{}, scope *gorm.Scope) (ChangedFields, error) {
	if x == nil || y == nil {
		return nil, errors.New("DiffRecord: arguments must not be nil")
	}
	// v1 is the fresh record (from API)
	v1 := reflect.ValueOf(x)
	// v2 is the cached record (from DB)
	v2 := reflect.ValueOf(y)
	if v1.Type() != v2.Type() {
		return nil, errors.New("DiffRecord: arguments are not the same type")
	}

	typ := v1.Type()
	if typ.Kind() != reflect.Struct {
		return nil, errors.New("DiffRecord: arguments must be structs")
	}

	var res ChangedFields
	for i, n := 0, v1.NumField(); i < n; i++ {
		f := typ.Field(i)
		fieldName := f.Name

		if strings.HasSuffix(fieldName, "ID") {
			// ignore
			continue
		}

		if f.Type.Kind() == reflect.Ptr {
			// ignore
			continue
		}

		if f.Type.Kind() == reflect.Slice {
			// ignore
			continue
		}

		if sf, ok := scope.FieldByName(fieldName); ok {
			if sf.IsIgnored {
				continue
			}
		} else {
			// not listed as a field? ignore
			continue
		}

		iseq, err := eq(v1.Field(i), v2.Field(i))
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		if !iseq {
			log.Printf("struct: field %d (%s) not equal", i, fieldName)
			if res == nil {
				res = make(ChangedFields)
			}
			res[fieldName] = v1.Field(i).Interface()
		}
	}
	return res, nil
}

// Comparison.
// Taken from text/template

var (
	errBadComparison = errors.New("incompatible types for comparison")
	errNoComparison  = errors.New("missing argument for comparison")
)

type kind int

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	stringKind
	uintKind
	timeKind
)

func basicKind(v reflect.Value) (kind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	case reflect.Struct:
		if _, ok := v.Interface().(time.Time); ok {
			return timeKind, nil
		}
	}
	return invalidKind, fmt.Errorf("bad type for comparison: %v", v.Type())
}

// eq evaluates the comparison a == b || a == c || ...
func eq(arg1 reflect.Value, arg2 ...reflect.Value) (bool, error) {
	v1 := indirectInterface(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}
	if len(arg2) == 0 {
		return false, errNoComparison
	}
	for _, arg := range arg2 {
		v2 := indirectInterface(arg)
		k2, err := basicKind(v2)
		if err != nil {
			return false, err
		}
		truth := false
		if k1 != k2 {
			// Special case: Can compare integer values regardless of type's sign.
			switch {
			case k1 == intKind && k2 == uintKind:
				truth = v1.Int() >= 0 && uint64(v1.Int()) == v2.Uint()
			case k1 == uintKind && k2 == intKind:
				truth = v2.Int() >= 0 && v1.Uint() == uint64(v2.Int())
			default:
				return false, errBadComparison
			}
		} else {
			switch k1 {
			case boolKind:
				truth = v1.Bool() == v2.Bool()
			case complexKind:
				truth = v1.Complex() == v2.Complex()
			case floatKind:
				truth = v1.Float() == v2.Float()
			case intKind:
				truth = v1.Int() == v2.Int()
			case stringKind:
				truth = v1.String() == v2.String()
			case uintKind:
				truth = v1.Uint() == v2.Uint()
			case timeKind:
				truth = v1.Interface().(time.Time) == v2.Interface().(time.Time)
			default:
				panic("invalid kind")
			}
		}
		if truth {
			return true, nil
		}
	}
	return false, nil
}

// indirectInterface returns the concrete value in an interface value,
// or else the zero reflect.Value.
// That is, if v represents the interface value x, the result is the same as reflect.ValueOf(x):
// the fact that x was an interface value is forgotten.
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem()
}
