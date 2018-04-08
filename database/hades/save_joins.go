package hades

import (
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func (c *Context) saveJoins(params *SaveParams, tx *gorm.DB, mtm *ManyToMany) error {
	partial := false
	for _, pj := range params.PartialJoins {
		if mtm.JoinTable == gorm.ToDBName(pj) {
			partial = true
		}
	}

	joinType := reflect.PtrTo(mtm.Scope.GetModelStruct().ModelType)

	getDestinKey := func(v reflect.Value) interface{} {
		return v.Elem().FieldByName(mtm.DestinName).Interface()
	}

	for sourceKey, joinRecs := range mtm.Values {
		cacheAddr := reflect.New(reflect.SliceOf(joinType))

		err := tx.Where(
			fmt.Sprintf(`"%s" = ?`, mtm.SourceDBName),
			sourceKey,
		).Find(cacheAddr.Interface()).Error
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

		if partial {
			// Not deleting extra join records, as requested
		} else {
			if len(deletes) > 0 {
				// FIXME: this needs to be paginated to avoid hitting SQLite max variables
				rec := reflect.New(joinType.Elem())
				err := tx.
					Delete(
						rec.Interface(),
						fmt.Sprintf(
							`"%s" = ? and "%s" in (?)`,
							mtm.SourceDBName,
							mtm.DestinDBName,
						),
						sourceKey,
						deletes,
					).Error
				if err != nil {
					return errors.Wrap(err, "deleting extraneous relations")
				}
			}
		}

		for _, joinRec := range inserts {
			rec := joinRec.Record

			if !rec.IsValid() {
				// if not passed an explicit record, make it ourselves
				// that typically means the join table doesn't have additional
				// columns and is a simple many2many
				rec = reflect.New(joinType.Elem())
				rec.Elem().FieldByName(mtm.SourceName).Set(reflect.ValueOf(sourceKey))
				rec.Elem().FieldByName(mtm.DestinName).Set(reflect.ValueOf(joinRec.DestinKey))
			}

			err := tx.Create(rec.Interface()).Error
			if err != nil {
				return errors.Wrap(err, "creating new relation records")
			}
		}

		for destinKey, rec := range updates {
			err := tx.Table(mtm.Scope.TableName()).
				Where(
					fmt.Sprintf(
						`"%s" = ? and "%s" = ?`,
						mtm.SourceDBName,
						mtm.DestinDBName,
					),
					sourceKey,
					destinKey,
				).Updates(rec).Error
			if err != nil {
				return errors.Wrap(err, "updating related records")
			}
		}
	}

	return nil
}
