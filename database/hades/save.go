package hades

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

type AllEntities map[reflect.Type]EntityMap
type EntityMap []interface{}

type SaveParams struct {
	// Record to save
	Record interface{}

	// Fields to save instead of the top-level record
	Assocs []string

	// Disable deleting join table entries (useful for partial data)
	PartialJoins []string
}

func (c *Context) Save(db *gorm.DB, params *SaveParams) error {
	return c.InTransaction(db, func(c *Context, tx *gorm.DB) error {
		return c.SaveNoTransaction(tx, params)
	})
}

func (c *Context) SaveNoTransaction(tx *gorm.DB, params *SaveParams) error {
	if params == nil {
		return errors.New("Save: params cannot be nil")
	}
	rec := params.Record
	assocs := params.Assocs
	consumer := c.Consumer

	startTime := time.Now()

	val := reflect.ValueOf(rec)

	riMap := make(RecordInfoMap)
	tree, err := c.WalkType(riMap, "<root>", val.Type(), make(VisitMap), assocs)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Debugf("record tree:\n%s", tree)

	entities := make(AllEntities)
	addEntity := func(v reflect.Value) error {
		typ := v.Type()
		if _, ok := c.ScopeMap[typ]; !ok {
			return fmt.Errorf("not a model type: %s", typ)
		}
		entities[typ] = append(entities[typ], v.Interface())
		return nil
	}

	var walk func(parent reflect.Value, v reflect.Value, ri *RecordInfo, persist bool) error

	var numVisited int64
	visit := func(parent reflect.Value, v reflect.Value, ri *RecordInfo, persist bool) error {
		ov := v

		if persist {
			if ri.Assoc != AssocNone {
				p := parent
				switch ri.Assoc {
				case AssocHasMany, AssocHasOne:
					pkField := p.Elem().FieldByName("ID")
					fkField := v.Elem().FieldByName(ri.ForeignKey)
					fkField.Set(pkField)
				case AssocBelongsTo:
					pkField := v.Elem().FieldByName("ID")
					fkField := p.Elem().FieldByName(ri.ForeignKey)
					fkField.Set(pkField)
				case AssocManyToMany:
					ri.ManyToMany.Add(p, v)
				}
			}

			numVisited++
			err := addEntity(v)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("expected a pointer, but got with %v", v)
		}
		v = v.Elem()

		if v.Kind() != reflect.Struct {
			return fmt.Errorf("expected a struct, but got with %v", v)
		}

		for _, child := range ri.Children {
			field := v.FieldByName(child.Name)
			if !field.IsValid() {
				continue
			}

			if field.Kind() == reflect.Ptr && field.IsNil() {
				continue
			}

			// always persist children
			err := walk(ov, field, child, true)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	walk = func(parent reflect.Value, v reflect.Value, ri *RecordInfo, persist bool) error {
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				err := visit(parent, v.Index(i), ri, persist)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			}
		} else {
			err := visit(parent, v, ri, persist)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	persistRoot := assocs == nil
	err = walk(reflect.Zero(reflect.TypeOf(0)), val, tree, persistRoot)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var modelNames []string
	for typ, m := range entities {
		consumer.Debugf("Found %d %s", len(m), typ)
		modelNames = append(modelNames, fmt.Sprintf("%v", typ))
	}
	consumer.Infof("Visited %d records (from %s) in %s", numVisited, strings.Join(modelNames, ", "), time.Since(startTime))

	startTime = time.Now()

	for _, m := range entities {
		err := c.saveRows(tx, params, m)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	for _, ri := range riMap {
		if ri.ManyToMany != nil {
			err := c.saveJoins(params, tx, ri.ManyToMany)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	consumer.Infof("Inserted %d, Updated %d, Deleted %d, Current %d in %s",
		c.Stats.Inserts,
		c.Stats.Updates,
		c.Stats.Deletes,
		c.Stats.Current,
		time.Since(startTime),
	)

	return nil
}
