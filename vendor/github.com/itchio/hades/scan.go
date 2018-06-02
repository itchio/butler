package hades

import (
	"reflect"
	"time"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

// ScanIntoRows is used to scan a single sqlite statement into a struct that
// might contain multiple models, for example
//
//   var c *hades.Context
//   var rows []struct{
//     Game Game `hades:"squash"`
//     CollectionGame `hades:"squash"`
//   }
//   // exec a query, and in the ResultFn:
//   c.Exec(..., ..., func () {
//     return c.ScanIntoRows(stmt, rows)
//   })
//
func (c *Context) ScanIntoRows(stmt *sqlite.Stmt, slicePtr interface{}) error {
	return c.IntoRowsScanner(slicePtr)(stmt)
}

// NewScannerIntoRows returns a ResultFn that scans all fields into what
// is typically an anonymous struct containing multiple squashed models.
// See ScanIntoRows.
func (c *Context) IntoRowsScanner(slicePtr interface{}) ResultFn {
	slicePtrVal := reflect.ValueOf(slicePtr)
	sliceVal := slicePtrVal.Elem()
	sliceTyp := sliceVal.Type()
	if sliceTyp.Kind() != reflect.Slice {
		err := errors.Errorf("ScanIntoRows expects a slice, got a %v", sliceTyp)
		return func(stmt *sqlite.Stmt) error {
			return err
		}
	}

	modelTyp := sliceTyp.Elem()
	recordTemplate := reflect.New(modelTyp).Elem()
	scope := c.NewScope(recordTemplate.Interface())

	return func(stmt *sqlite.Stmt) error {
		recordVal := reflect.New(modelTyp).Elem()
		err := c.Scan(stmt, scope.GetStructFields(), recordVal)
		if err != nil {
			return err
		}
		sliceVal.Set(reflect.Append(sliceVal, recordVal))
		return nil
	}
}

// Scan is used to scan a single sqlite statement into a model struct
// for example:
//
//   var c *hades.Context
//   var g Game
//   c.Scan(stmt, c.NewScope(), reflect.ValueOf(&g).Elem())
//
func (c *Context) Scan(stmt *sqlite.Stmt, structFields []*StructField, result reflect.Value) error {
	i := 0

	var processField func(sf *StructField, result reflect.Value) error
	processField = func(sf *StructField, result reflect.Value) error {
		field := result.FieldByName(sf.Name)
		if sf.IsSquashed {
			for _, nsf := range sf.SquashedFields {
				err := processField(nsf, field)
				if err != nil {
					return err
				}
			}
			return nil
		}

		if !sf.IsNormal {
			return nil
		}

		fieldEl := field
		typ := field.Type()
		wasPtr := false

		colTyp := stmt.ColumnType(i)

		if typ.Kind() == reflect.Ptr {
			wasPtr = true
			if colTyp == sqlite.SQLITE_NULL {
				field.Set(reflect.Zero(field.Type()))
				i++
				return nil
			}

			fieldEl = field.Elem()
			typ = typ.Elem()
		}

		switch typ.Kind() {
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int,
			reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:

			val := stmt.ColumnInt64(i)
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetInt(val)
			}
		case reflect.Float64, reflect.Float32:
			val := stmt.ColumnFloat(i)
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetFloat(val)
			}
		case reflect.Bool:
			val := stmt.ColumnInt(i) == 1
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetBool(val)
			}
		case reflect.String:
			val := stmt.ColumnText(i)
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetString(val)
			}
		case reflect.Struct:
			if typ == reflect.TypeOf(time.Time{}) {
				text := stmt.ColumnText(i)
				tim, err := time.Parse(time.RFC3339Nano, text)
				if err == nil {
					if wasPtr {
						field.Set(reflect.ValueOf(&tim))
					} else {
						fieldEl.Set(reflect.ValueOf(tim))
					}
				}
				break
			}
			fallthrough
		default:
			return errors.Errorf("For model %s, unknown kind %s for field %s", result.Type(), field.Type().Kind(), sf.Name)
		}

		i++
		return nil
	}

	for _, sf := range structFields {
		err := processField(sf, result)
		if err != nil {
			return err
		}
	}

	return nil
}
