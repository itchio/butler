package hades

import (
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

func (c *Context) saveRows(tx *gorm.DB, params *SaveParams, inputIface interface{}) error {
	tx = tx.Set("gorm:save_associations", false)
	consumer := c.Consumer

	// inputIFace is a `[]interface{}`
	input := reflect.ValueOf(inputIface)
	if input.Kind() != reflect.Slice {
		return errors.New("diff needs a slice")
	}

	if input.Len() == 0 {
		consumer.Infof("Nothing to persist (0 input records)")
		return nil
	}

	// we're trying to make fresh a `[]*SomeModel` slice, so
	// that gorm can get annotation information from SomeModel struct annotations
	first := input.Index(0).Elem()
	fresh := reflect.MakeSlice(reflect.SliceOf(first.Type()), input.Len(), input.Len())
	for i := 0; i < input.Len(); i++ {
		record := input.Index(i).Elem()
		fresh.Index(i).Set(record)
	}

	// use gorm facilities to find the primary keys
	scope := tx.NewScope(first.Interface())
	modelName := scope.GetModelStruct().ModelType.Name()
	var pkFields []*gorm.Field
	var pkColumnNames []string
	fs := scope.Fields()
	for _, f := range fs {
		if f.IsPrimaryKey {
			pkFields = append(pkFields, f)
			pkColumnNames = append(pkColumnNames, f.DBName)
		}
	}

	consumer.Debugf("Persisting %d records for %s, primary keys: (%s)", fresh.Len(), modelName, strings.Join(pkColumnNames, ", "))

	// this will happen for associations, we should have another codepath for that
	if len(pkFields) != 1 {
		if len(pkFields) != 2 {
			return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(pkFields), modelName)
		}

		bfm := make(map[*gorm.Field]map[interface{}][]reflect.Value)

		for _, pkf := range pkFields {
			rm := make(map[interface{}][]reflect.Value)

			for i := 0; i < fresh.Len(); i++ {
				rec := fresh.Index(i)
				v := rec.Elem().FieldByName(pkf.Name).Interface()
				rm[v] = append(rm[v], rec)
			}
			bfm[pkf] = rm
		}

		var bestLPK *gorm.Field
		var bestNumGroups int64 = math.MaxInt64
		var valueMap map[interface{}][]reflect.Value
		for pk, recs := range bfm {
			numGroups := len(recs)
			if int64(numGroups) < bestNumGroups {
				bestLPK = pk
				bestNumGroups = int64(numGroups)
				valueMap = recs
			}
		}

		if bestLPK == nil {
			return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(pkFields), modelName)
		}

		var bestRPK *gorm.Field
		for pk, _ := range bfm {
			if pk != bestLPK {
				bestRPK = pk
				break
			}
		}
		if bestRPK == nil {
			return errors.New("Internal error: could not find bestRPK")
		}

		LPKModelName := strings.TrimSuffix(bestLPK.Name, "ID")
		RPKModelName := strings.TrimSuffix(bestRPK.Name, "ID")
		var LPKScope, RPKScope *gorm.Scope

		for _, s := range c.ScopeMap {
			if s.GetModelStruct().ModelType.Name() == LPKModelName {
				LPKScope = s
			}
			if s.GetModelStruct().ModelType.Name() == RPKModelName {
				RPKScope = s
			}
		}

		if LPKScope == nil {
			return fmt.Errorf("Could not find LPKModel %s", LPKModelName)
		}
		if RPKScope == nil {
			return fmt.Errorf("Could not find RPKModel %s", RPKModelName)
		}

		if len(LPKScope.PrimaryFields()) != 1 {
			return fmt.Errorf("Expected LPK model %s to have 1 primary field, but it has %d", LPKModelName, len(LPKScope.PrimaryFields()))
		}
		if len(RPKScope.PrimaryFields()) != 1 {
			return fmt.Errorf("Expected RPK model %s to have 1 primary field, but it has %d", LPKModelName, len(RPKScope.PrimaryFields()))
		}

		mtm, err := c.NewManyToMany(
			scope.TableName(),
			[]gorm.JoinTableForeignKey{
				gorm.JoinTableForeignKey{
					DBName:            gorm.ToDBName(bestLPK.Name),
					AssociationDBName: LPKScope.PrimaryField().DBName,
				},
			},
			[]gorm.JoinTableForeignKey{
				gorm.JoinTableForeignKey{
					DBName:            gorm.ToDBName(bestRPK.Name),
					AssociationDBName: LPKScope.PrimaryField().DBName,
				},
			},
		)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		for lpk, recs := range valueMap {
			for _, rec := range recs {
				rpk := rec.Elem().FieldByName(bestRPK.Name).Interface()
				mtm.AddKeys(lpk, rpk, rec)
			}
		}

		err = c.saveJoins(params, tx, mtm)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	pkField := pkFields[0]

	// record should be a *SomeModel, we're effectively doing (*record).<pkColumn>
	getPk := func(record reflect.Value) interface{} {
		f := record.Elem().FieldByName(pkField.Name)
		if !f.IsValid() {
			return nil
		}
		return f.Interface()
	}

	// collect primary key values for all of input
	var pks []interface{}
	for i := 0; i < fresh.Len(); i++ {
		record := fresh.Index(i)
		pks = append(pks, getPk(record))
	}

	var err error

	// retrieve cached items in a []*SomeModel
	// for some reason, reflect.New returns a &[]*SomeModel instead,
	// I'm guessing slices can't be interfaces, but pointers to slices can?
	pagedByPK := func(pkDBName string, pks []interface{}, sliceType reflect.Type) (reflect.Value, error) {
		// actually defaults to 999, but let's get some breathing room
		const maxSqlVars = 900
		result := reflect.New(sliceType)
		resultVal := result.Elem()

		remainingItems := pks
		consumer.Debugf("%d items to fetch by PK", len(pks))

		for len(remainingItems) > 0 {
			var pageSize int
			if len(remainingItems) > maxSqlVars {
				pageSize = maxSqlVars
			} else {
				pageSize = len(remainingItems)
			}

			consumer.Debugf("Fetching %d items", pageSize)
			pageAddr := reflect.New(sliceType)
			err = tx.Where(fmt.Sprintf("%s in (?)", pkDBName), remainingItems[:pageSize]).Find(pageAddr.Interface()).Error
			if err != nil {
				return result, errors.Wrap(err, 0)
			}

			appended := reflect.AppendSlice(resultVal, pageAddr.Elem())
			resultVal.Set(appended)
			remainingItems = remainingItems[pageSize:]
		}

		return result, nil
	}

	cacheAddr, err := pagedByPK(pkField.DBName, pks, fresh.Type())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cache := cacheAddr.Elem()

	// index cached items by their primary key
	// so we can look them up in O(1) when comparing
	cacheByPK := make(map[interface{}]reflect.Value)
	for i := 0; i < cache.Len(); i++ {
		record := cache.Index(i)
		cacheByPK[getPk(record)] = record
	}

	// compare cached records with fresh records
	var inserts []reflect.Value
	var updates = make(map[interface{}]ChangedFields)

	donePK := make(map[interface{}]bool)
	for i := 0; i < fresh.Len(); i++ {
		frec := fresh.Index(i)
		pk := getPk(frec)
		if _, ok := donePK[pk]; ok {
			continue
		}
		donePK[pk] = true

		if crec, ok := cacheByPK[pk]; ok {
			// frec and crec are *SomeModel, but `RecordEqual` ignores pointer
			// equality - we want to compare the contents of the struct
			// so we indirect to SomeModel here.
			ifrec := frec.Elem().Interface()
			icrec := crec.Elem().Interface()

			cf, err := DiffRecord(ifrec, icrec, scope)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			if cf != nil {
				updates[pk] = cf
			}
		} else {
			inserts = append(inserts, frec)
		}
	}

	c.Stats.Inserts += int64(len(inserts))
	c.Stats.Updates += int64(len(updates))
	c.Stats.Current += int64(fresh.Len() - len(updates) - len(inserts))

	if len(inserts) > 0 {
		for _, rec := range inserts {
			err := tx.Create(rec.Interface()).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	for pk, rec := range updates {
		err := tx.Table(scope.TableName()).
			Where(
				fmt.Sprintf(
					`"%s" = ?`,
					pkField.DBName,
				),
				pk,
			).
			Updates(rec).
			Error
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}
