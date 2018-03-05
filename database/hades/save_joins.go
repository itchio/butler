package hades

import (
	"fmt"
	"reflect"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

func (c *Context) saveJoins(params *SaveParams, tx *gorm.DB, mtm *ManyToMany) error {
	consumer := c.Consumer

	partial := false
	for _, pj := range params.PartialJoins {
		if mtm.Scope.TableName() == gorm.ToDBName(pj) {
			consumer.Debugf("Saving partial joins for %s", mtm.Scope.TableName())
			partial = true
		}
	}

	joinType := reflect.PtrTo(mtm.Scope.GetModelStruct().ModelType)

	getRpk := func(v reflect.Value) interface{} {
		// TODO: handle different PKs
		return v.Elem().FieldByName(mtm.RPKColumn).Interface()
	}

	for lpk, joinRecs := range mtm.Values {
		cacheAddr := reflect.New(reflect.SliceOf(joinType))

		err := tx.Where(
			fmt.Sprintf(`"%s" = ?`, gorm.ToDBName(mtm.LPKColumn)),
			lpk,
		).
			Find(cacheAddr.Interface()).Error
		if err != nil {
			return errors.Wrap(err, 0)
		}

		cache := cacheAddr.Elem()

		cacheByRPK := make(map[interface{}]reflect.Value)
		for i := 0; i < cache.Len(); i++ {
			rec := cache.Index(i)
			cacheByRPK[getRpk(rec)] = rec
		}

		freshByRPK := make(map[interface{}]reflect.Value)
		for _, joinRec := range joinRecs {
			freshByRPK[joinRec.RPK] = joinRec.Record
		}

		var deletes []interface{}
		updates := make(map[interface{}]ChangedFields)
		var inserts []JoinRec

		// compare with cache: will result in delete or update
		for i := 0; i < cache.Len(); i++ {
			crec := cache.Index(i)
			rpk := getRpk(crec)
			if frec, ok := freshByRPK[rpk]; ok {
				if frec.IsValid() {
					// compare to maybe update
					ifrec := frec.Elem().Interface()
					icrec := crec.Elem().Interface()

					cf, err := DiffRecord(ifrec, icrec, mtm.Scope)
					if err != nil {
						return errors.Wrap(err, 0)
					}

					if cf != nil {
						updates[rpk] = cf
					}
				}
			} else {
				deletes = append(deletes, rpk)
			}
		}

		for _, joinRec := range joinRecs {
			if _, ok := cacheByRPK[joinRec.RPK]; !ok {
				inserts = append(inserts, joinRec)
			}
		}

		consumer.Debugf("SaveJoins: %d Inserts, %d Updates, %d Deletes", len(inserts), len(updates), len(deletes))

		if partial {
			// Not deleting extra join records, as requested
		} else {
			if len(deletes) > 0 {
				rec := reflect.New(joinType.Elem())
				err := tx.
					Delete(
						rec.Interface(),
						fmt.Sprintf(
							`"%s" = ? and "%s" in (?)`,
							gorm.ToDBName(mtm.LPKColumn),
							gorm.ToDBName(mtm.RPKColumn),
						),
						lpk,
						deletes,
					).Error
				if err != nil {
					return errors.Wrap(err, 0)
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
				rec.Elem().FieldByName(mtm.LPKColumn).Set(reflect.ValueOf(lpk))
				rec.Elem().FieldByName(mtm.RPKColumn).Set(reflect.ValueOf(joinRec.RPK))
			}

			err := tx.Create(rec.Interface()).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		for rpk, rec := range updates {
			err := tx.Table(mtm.Scope.TableName()).
				Where(
					fmt.Sprintf(
						`"%s" = ? and "%s" = ?`,
						gorm.ToDBName(mtm.LPKColumn),
						gorm.ToDBName(mtm.RPKColumn),
					),
					lpk,
					rpk,
				).Updates(rec).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	return nil
}
