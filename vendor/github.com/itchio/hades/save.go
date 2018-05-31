package hades

import (
	"fmt"
	"reflect"

	"github.com/go-xorm/builder"
	"github.com/itchio/hades/sqliteutil2"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

type AllEntities map[reflect.Type]EntityMap
type EntityMap []interface{}

type SaveParams struct {
	// Record to save
	Record interface{}

	// Fields to save instead of the top-level record
	Assocs []string

	// For has_many and many_to_many, never delete rows for these models
	DontCull []interface{}
}

func (c *Context) SaveOne(conn *sqlite.Conn, record interface{}) (err error) {
	return c.SaveNoTransaction(conn, &SaveParams{
		Record: record,
	})
}

func (c *Context) Save(conn *sqlite.Conn, params *SaveParams) (err error) {
	defer sqliteutil2.Save(conn)(&err)

	return c.SaveNoTransaction(conn, params)
}

func (c *Context) SaveNoTransaction(conn *sqlite.Conn, params *SaveParams) error {
	if params == nil {
		return errors.New("Save: params cannot be nil")
	}
	rec := params.Record
	assocs := params.Assocs

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

	entities := make(AllEntities)
	addEntity := func(v reflect.Value) error {
		typ := v.Type()
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
			cull := false

			if vri.Relationship != nil {
				switch vri.Relationship.Kind {
				case "has_many":
					cull = true
					for _, dc := range params.DontCull {
						if reflect.TypeOf(dc).Elem() == vri.ModelStruct.ModelType {
							cull = false
						}
					}
				case "many_to_many":
					// culling is done later, but let's record the ManyToMany now
					vri.ManyToMany.Mark(p)
				}
			}

			for i := 0; i < v.Len(); i++ {
				err := visit(p, pri, v.Index(i), vri, persist)
				if err != nil {
					return errors.Wrap(err, "walking slice of children")
				}
			}

			if cull {
				var oldValuePKs []string
				rel := vri.Relationship

				parentPF := c.NewScope(p.Interface()).PrimaryField()
				if parentPF == nil {
					return errors.Errorf("Can't save %v has_many %v: parent has no primary keys", pri.Type, vri.Type)
				}
				parentPK := parentPF.Field

				if len(vri.ModelStruct.PrimaryFields) != 1 {
					return errors.Errorf("Since %v has_many %v", pri.Name, vri.Name)
				}
				valuePF := c.NewScope(v.Interface()).PrimaryField()
				if valuePF == nil {
					return errors.Errorf("Can't save %v has_many %v: value has no primary keys", pri.Type, vri.Type)
				}

				q := builder.Select(rel.AssociationForeignDBNames[0]).
					From(vri.ModelStruct.TableName).
					Where(builder.Eq{
						rel.ForeignDBNames[0]: parentPK,
					})

				err = c.Exec(conn, q, func(stmt *sqlite.Stmt) error {
					pk := stmt.ColumnText(0)
					oldValuePKs = append(oldValuePKs, pk)
					return nil
				})
				if err != nil {
					return err
				}

				if len(oldValuePKs) > 0 {
					var newValuePKs []string
					for i := 0; i < v.Len(); i++ {
						newValuePKs = append(newValuePKs, c.NewScope(v.Index(i).Interface()).PrimaryField().Field.String())
					}

					var newValuePKsMap = make(map[string]struct{})
					for _, pk := range newValuePKs {
						newValuePKsMap[pk] = struct{}{}
					}

					var vpksToDelete []interface{}
					for _, pk := range oldValuePKs {
						if _, ok := newValuePKsMap[pk]; !ok {
							vpksToDelete = append(vpksToDelete, pk)
						}
					}

					if len(vpksToDelete) > 0 {
						err := c.deletePagedByPK(conn, vri.ModelStruct.TableName, valuePF.DBName, vpksToDelete, builder.NewCond())
						if err != nil {
							return err
						}
					}
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

	for _, m := range entities {
		err := c.saveRows(conn, params, m)
		if err != nil {
			return errors.Wrap(err, "saving rows")
		}
	}

	for _, ri := range riMap {
		if ri.ManyToMany != nil {
			err := c.saveJoins(params, conn, ri.ManyToMany)
			if err != nil {
				return errors.Wrap(err, "saving joins")
			}
		}
	}

	return nil
}
