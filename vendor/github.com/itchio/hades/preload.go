package hades

import (
	"fmt"
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

func (c *Context) Preload(conn *sqlite.Conn, rec interface{}, opts ...PreloadParam) error {
	params := &preloadParams{}
	for _, o := range opts {
		o.ApplyToPreloadParams(params)
	}

	if len(params.assocs) == 0 {
		return errors.Errorf("Cannot preload 0 assocs")
	}

	val := reflect.ValueOf(rec)
	valtyp := val.Type()
	if valtyp.Kind() == reflect.Slice {
		if val.Len() == 0 {
			return nil
		}
		valtyp = valtyp.Elem()
	}
	if valtyp.Kind() != reflect.Ptr {
		return errors.Errorf("Preload expects a []*Model or *Model, but it was passed a %v instead", val.Type())
	}

	riMap := make(RecordInfoMap)
	rootField := &assocField{
		name:     fmt.Sprintf("%v", valtyp),
		mode:     AssocModeAppend,
		children: params.assocs,
	}
	rootInfo, err := c.WalkType(riMap, rootField, valtyp)
	if err != nil {
		return errors.WithMessage(err, "waking type tree")
	}

	var walk func(p reflect.Value, pri *RecordInfo) error
	walk = func(p reflect.Value, pri *RecordInfo) error {
		ptyp := p.Type()
		if ptyp.Kind() == reflect.Slice {
			ptyp = ptyp.Elem()
		}
		if ptyp.Kind() != reflect.Ptr {
			return errors.Errorf("walk expects a []*Model or *Model, but it was passed a %v instead", p.Type())
		}

		for _, cri := range pri.Children {
			freshAddr := reflect.New(reflect.SliceOf(cri.Type))

			var ps reflect.Value
			if p.Type().Kind() == reflect.Slice {
				ps = p
			} else {
				ps = reflect.MakeSlice(reflect.SliceOf(p.Type()), 1, 1)
				ps.Index(0).Set(p)
			}

			switch cri.Relationship.Kind {
			case "has_many":
				var keys []interface{}
				for i := 0; i < ps.Len(); i++ {
					keys = append(keys, ps.Index(i).Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface())
				}

				var err error
				freshAddr, err = c.fetchPagedByPK(conn, cri.Relationship.ForeignDBNames[0], keys, reflect.SliceOf(cri.Type), cri.Field.Search())
				if err != nil {
					return errors.WithMessage(err, "fetching has_many records (paginated)")
				}

				pByFK := make(map[interface{}]reflect.Value)
				for i := 0; i < ps.Len(); i++ {
					rec := ps.Index(i)
					fk := rec.Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface()
					pByFK[fk] = rec

					// reset slices so if preload is called more than once,
					// it doesn't keep appending
					field := rec.Elem().FieldByName(cri.Name())
					field.Set(reflect.New(field.Type()).Elem())
				}

				fresh := freshAddr.Elem()
				for i := 0; i < fresh.Len(); i++ {
					fk := fresh.Index(i).Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface()
					if p, ok := pByFK[fk]; ok {
						dest := p.Elem().FieldByName(cri.Name())
						dest.Set(reflect.Append(dest, fresh.Index(i)))
					}
				}
			case "has_one":
				// child (c, cri) has a parent_id field (p)
				var keys []interface{}
				for i := 0; i < ps.Len(); i++ {
					keys = append(keys, ps.Index(i).Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface())
				}

				var err error
				freshAddr, err = c.fetchPagedByPK(conn, cri.Relationship.ForeignDBNames[0], keys, reflect.SliceOf(cri.Type), cri.Field.Search())
				if err != nil {
					return errors.WithMessage(err, "fetching has_one records (paginated)")
				}

				fresh := freshAddr.Elem()
				freshByFK := make(map[interface{}]reflect.Value)
				for i := 0; i < fresh.Len(); i++ {
					rec := fresh.Index(i)
					fk := rec.Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface()
					freshByFK[fk] = rec
				}

				for i := 0; i < ps.Len(); i++ {
					prec := ps.Index(i)
					fk := prec.Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface()
					if crec, ok := freshByFK[fk]; ok {
						prec.Elem().FieldByName(cri.Name()).Set(crec)
					}
				}
			case "belongs_to":
				// parent (p) has a child_id field (c, cri)
				var keys []interface{}
				for i := 0; i < ps.Len(); i++ {
					keys = append(keys, ps.Index(i).Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface())
				}

				var err error
				freshAddr, err = c.fetchPagedByPK(conn, cri.Relationship.AssociationForeignDBNames[0], keys, reflect.SliceOf(cri.Type), cri.Field.Search())
				if err != nil {
					return errors.WithMessage(err, "fetching belongs_to records (paginated)")
				}

				fresh := freshAddr.Elem()
				freshByFK := make(map[interface{}]reflect.Value)
				for i := 0; i < fresh.Len(); i++ {
					rec := fresh.Index(i)
					fk := rec.Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface()
					freshByFK[fk] = rec
				}

				for i := 0; i < ps.Len(); i++ {
					prec := ps.Index(i)
					fk := prec.Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface()
					if crec, ok := freshByFK[fk]; ok {
						prec.Elem().FieldByName(cri.Name()).Set(crec)
					}
				}
			default:
				return errors.Errorf("Preload doesn't know how to handle %s relationships", cri.Relationship.Kind)
			}

			fresh := freshAddr.Elem()

			err = walk(fresh, cri)
			if err != nil {
				return errors.WithStack(err)
			}
		}
		return nil
	}
	err = walk(val, rootInfo)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}
