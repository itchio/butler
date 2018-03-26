package hades

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
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
	valtyp := val.Type()
	if valtyp.Kind() == reflect.Slice {
		valtyp = valtyp.Elem()
	}
	if valtyp.Kind() != reflect.Ptr {
		return fmt.Errorf("Save expects a []*Model or a *Model, but it was passed a %v instead", val.Type())
	}

	riMap := make(RecordInfoMap)
	tree, err := c.WalkType(riMap, "<root>", valtyp, make(VisitMap), assocs)
	if err != nil {
		return errors.Wrap(err, "walking records to be saved")
	}
	consumer.Debugf("RecordInfo tree:\n%s", tree)

	entities := make(AllEntities)
	addEntity := func(v reflect.Value) error {
		typ := v.Type()
		if _, ok := c.ScopeMap[typ]; !ok {
			return fmt.Errorf("not a model type: %s", typ)
		}
		entities[typ] = append(entities[typ], v.Interface())
		return nil
	}

	var walk func(p reflect.Value, pri *RecordInfo, v reflect.Value, vri *RecordInfo, persist bool) error

	var numVisited int64
	visit := func(p reflect.Value, pri *RecordInfo, v reflect.Value, vri *RecordInfo, persist bool) error {
		if persist {
			if vri.Relationship != nil {
				switch vri.Relationship.Kind {
				case "has_many", "has_one":
					if len(pri.ModelStruct.PrimaryFields) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one primary key in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							p.Type(),
							len(pri.ModelStruct.PrimaryFields),
						)
					}
					pkField := p.Elem().FieldByName(pri.ModelStruct.PrimaryFields[0].Name)
					if len(vri.Relationship.ForeignFieldNames) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one foreign field in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							v.Type(),
							len(vri.Relationship.ForeignFieldNames),
						)
					}
					fkField := v.Elem().FieldByName(vri.Relationship.ForeignFieldNames[0])
					fkField.Set(pkField)
				case "belongs_to":
					if len(vri.ModelStruct.PrimaryFields) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one primary key in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							v.Type(),
							len(vri.ModelStruct.PrimaryFields),
						)
					}
					pkField := v.Elem().FieldByName(vri.ModelStruct.PrimaryFields[0].Name)

					if len(vri.Relationship.ForeignFieldNames) != 1 {
						return fmt.Errorf("Since %v %s %v, we expected one foreign field in %v, but found %d",
							p.Type(),
							vri.Relationship.Kind,
							v.Type(),
							v.Type(),
							len(vri.Relationship.ForeignFieldNames),
						)
					}
					fkField := p.Elem().FieldByName(vri.Relationship.ForeignFieldNames[0])
					fkField.Set(pkField)
				case "many_to_many":
					vri.ManyToMany.Add(p, v)
				}
			}

			numVisited++
			err := addEntity(v)
			if err != nil {
				return errors.Wrap(err, "adding entity")
			}
		}

		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("expected a pointer, but got with %v", v)
		}
		vs := v.Elem()

		if vs.Kind() != reflect.Struct {
			return fmt.Errorf("expected a struct, but got with %v", v)
		}

		for _, childRi := range vri.Children {
			child := vs.FieldByName(childRi.Name)
			if !child.IsValid() {
				continue
			}

			if child.Kind() == reflect.Ptr && child.IsNil() {
				continue
			}

			// children are always saved
			persistChildren := true
			err := walk(v, vri, child, childRi, persistChildren)
			if err != nil {
				return errors.Wrap(err, "walking child entities to be saved")
			}
		}
		return nil
	}

	walk = func(p reflect.Value, pri *RecordInfo, v reflect.Value, vri *RecordInfo, persist bool) error {
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				err := visit(p, pri, v.Index(i), vri, persist)
				if err != nil {
					return errors.Wrap(err, "walking slice of children")
				}
			}
		} else {
			err := visit(p, pri, v, vri, persist)
			if err != nil {
				return errors.Wrap(err, "walking single child")
			}
		}
		return nil
	}

	persistRoot := assocs == nil
	err = walk(reflect.Zero(reflect.TypeOf(0)), nil, val, tree, persistRoot)
	if err != nil {
		return errors.Wrap(err, "walking all records to be persisted")
	}

	var modelNames []string
	for typ, m := range entities {
		consumer.Debugf("Found %d %s", len(m), typ)
		modelNames = append(modelNames, fmt.Sprintf("%v", typ))
	}
	consumer.Debugf("Visited %d records (from %s) in %s", numVisited, strings.Join(modelNames, ", "), time.Since(startTime))

	startTime = time.Now()

	for _, m := range entities {
		err := c.saveRows(tx, params, m)
		if err != nil {
			return errors.Wrap(err, "saving rows")
		}
	}

	for _, ri := range riMap {
		if ri.ManyToMany != nil {
			err := c.saveJoins(params, tx, ri.ManyToMany)
			if err != nil {
				return errors.Wrap(err, "saving joins")
			}
		}
	}

	consumer.Debugf("Inserted %d, Updated %d, Deleted %d, Current %d in %s",
		c.Stats.Inserts,
		c.Stats.Updates,
		c.Stats.Deletes,
		c.Stats.Current,
		time.Since(startTime),
	)

	return nil
}
