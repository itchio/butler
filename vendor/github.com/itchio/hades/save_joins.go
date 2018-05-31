package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/pkg/errors"
)

func (c *Context) saveJoins(params *SaveParams, conn *sqlite.Conn, mtm *ManyToMany) error {
	cull := true
	for _, dc := range params.DontCull {
		if mtm.JoinTable == ToDBName(c.NewScope(dc).TableName()) {
			cull = false
			break
		}
	}

	joinType := reflect.PtrTo(mtm.Scope.GetModelStruct().ModelType)

	getDestinKey := func(v reflect.Value) interface{} {
		return v.Elem().FieldByName(mtm.DestinName).Interface()
	}

	for sourceKey, joinRecs := range mtm.Values {
		cacheAddr := reflect.New(reflect.SliceOf(joinType))

		err := c.Select(conn, cacheAddr.Interface(), builder.Eq{mtm.SourceDBName: sourceKey}, nil)
		if err != nil {
			return errors.Wrap(err, "fetching cached records to compare later")
		}

		cache := cacheAddr.Elem()

		cacheByDestinKey := make(map[interface{}]reflect.Value)
		for i := 0; i < cache.Len(); i++ {
			rec := cache.Index(i)
			cacheByDestinKey[getDestinKey(rec)] = rec
		}

		freshByDestinKey := make(map[interface{}]reflect.Value)
		for _, joinRec := range joinRecs {
			freshByDestinKey[joinRec.DestinKey] = joinRec.Record
		}

		var deletes []interface{}
		updates := make(map[interface{}]ChangedFields)
		var inserts []JoinRec

		// compare with cache: will result in delete or update
		for i := 0; i < cache.Len(); i++ {
			crec := cache.Index(i)
			destinKey := getDestinKey(crec)
			if frec, ok := freshByDestinKey[destinKey]; ok {
				if frec.IsValid() {
					// compare to maybe update
					ifrec := frec.Elem().Interface()
					icrec := crec.Elem().Interface()

					cf, err := DiffRecord(ifrec, icrec, mtm.Scope)
					if err != nil {
						return errors.Wrap(err, "diffing database records")
					}

					if cf != nil {
						updates[destinKey] = cf
					}
				}
			} else {
				deletes = append(deletes, destinKey)
			}
		}

		for _, joinRec := range joinRecs {
			if _, ok := cacheByDestinKey[joinRec.DestinKey]; !ok {
				inserts = append(inserts, joinRec)
			}
		}

		if !cull {
			// Not deleting extra join records, as requested
		} else {
			if len(deletes) > 0 {
				err := c.deletePagedByPK(conn, mtm.JoinTable, mtm.DestinDBName, deletes, builder.Eq{mtm.SourceDBName: sourceKey})
				if err != nil {
					return errors.Wrap(err, "deleting extraneous relations")
				}
			}
		}

		for _, joinRec := range inserts {
			rec := joinRec.Record

			if rec.IsValid() {
				err := c.Insert(conn, mtm.Scope, rec)
				if err != nil {
					return errors.Wrap(err, "creating new relation records")
				}
			} else {
				// if not passed an explicit record, make it ourselves
				// that typically means the join table doesn't have additional
				// columns and is a simple many_to_many
				eq := builder.Eq{
					mtm.SourceDBName: sourceKey,
					mtm.DestinDBName: joinRec.DestinKey,
				}
				query := builder.Insert(eq).Into(mtm.JoinTable)
				err := c.Exec(conn, query, nil)
				if err != nil {
					return err
				}
			}
		}

		for destinKey, cf := range updates {
			query := builder.Update(cf.ToEq()).Into(mtm.Scope.TableName()).Where(builder.Eq{mtm.SourceDBName: sourceKey, mtm.DestinDBName: destinKey})
			err := c.Exec(conn, query, nil)
			if err != nil {
				return errors.Wrap(err, "updating related records")
			}
		}
	}

	return nil
}
