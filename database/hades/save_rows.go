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
		consumer.Debugf("Nothing to persist (0 input records)")
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
	primaryFields := scope.PrimaryFields()

	consumer.Debugf("Persisting %d records for %s", fresh.Len(), modelName)

	// this will happen for associations
	if len(primaryFields) != 1 {
		if len(primaryFields) != 2 {
			return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(primaryFields), modelName)
		}

		recordsGroupedByPrimaryField := make(map[*gorm.Field]map[interface{}][]reflect.Value)

		for _, primaryField := range primaryFields {
			recordsByKey := make(map[interface{}][]reflect.Value)

			for i := 0; i < fresh.Len(); i++ {
				rec := fresh.Index(i)
				key := rec.Elem().FieldByName(primaryField.Name).Interface()
				recordsByKey[key] = append(recordsByKey[key], rec)
			}
			recordsGroupedByPrimaryField[primaryField] = recordsByKey
		}

		var bestSourcePrimaryField *gorm.Field
		var bestNumGroups int64 = math.MaxInt64
		var valueMap map[interface{}][]reflect.Value
		for primaryField, recs := range recordsGroupedByPrimaryField {
			numGroups := len(recs)
			if int64(numGroups) < bestNumGroups {
				bestSourcePrimaryField = primaryField
				bestNumGroups = int64(numGroups)
				valueMap = recs
			}
		}

		if bestSourcePrimaryField == nil {
			return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(primaryFields), modelName)
		}

		var bestDestinPrimaryField *gorm.Field
		for primaryField, _ := range recordsGroupedByPrimaryField {
			if primaryField != bestSourcePrimaryField {
				bestDestinPrimaryField = primaryField
				break
			}
		}
		if bestDestinPrimaryField == nil {
			return errors.New("Internal error: could not find bestDestinPrimaryField")
		}

		sourceRelField, ok := scope.FieldByName(strings.TrimSuffix(bestSourcePrimaryField.Name, "ID"))
		if !ok {
			return fmt.Errorf("Could not find assoc for %s.%s", modelName, bestSourcePrimaryField.Name)
		}
		destinRelField, ok := scope.FieldByName(strings.TrimSuffix(bestDestinPrimaryField.Name, "ID"))
		if !ok {
			return fmt.Errorf("Could not find assoc for %s.%s", modelName, bestDestinPrimaryField.Name)
		}

		sourceScope, ok := c.ScopeMap[sourceRelField.Struct.Type]
		if !ok {
			return fmt.Errorf("Could not find scope for assoc for %s.%s", modelName, bestSourcePrimaryField.Name)
		}
		destinScope, ok := c.ScopeMap[destinRelField.Struct.Type]
		if !ok {
			return fmt.Errorf("Could not find scope for assoc for %s.%s", modelName, bestSourcePrimaryField.Name)
		}

		if len(sourceScope.PrimaryFields()) != 1 {
			return fmt.Errorf("Expected Source model %s to have 1 primary field, but it has %d",
				sourceScope.GetModelStruct().ModelType, len(sourceScope.PrimaryFields()))
		}
		if len(destinScope.PrimaryFields()) != 1 {
			return fmt.Errorf("Expected Destin model %s to have 1 primary field, but it has %d",
				destinScope.GetModelStruct().ModelType, len(destinScope.PrimaryFields()))
		}

		sourceJTFK := gorm.JoinTableForeignKey{
			DBName:            gorm.ToDBName(bestSourcePrimaryField.Name),
			AssociationDBName: sourceScope.PrimaryField().DBName,
		}

		destinJTFK := gorm.JoinTableForeignKey{
			DBName:            gorm.ToDBName(bestDestinPrimaryField.Name),
			AssociationDBName: destinScope.PrimaryField().DBName,
		}

		mtm, err := c.NewManyToMany(
			scope.TableName(),
			[]gorm.JoinTableForeignKey{sourceJTFK},
			[]gorm.JoinTableForeignKey{destinJTFK},
		)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		for sourceKey, recs := range valueMap {
			for _, rec := range recs {
				destinKey := rec.Elem().FieldByName(bestDestinPrimaryField.Name).Interface()
				mtm.AddKeys(sourceKey, destinKey, rec)
			}
		}

		err = c.saveJoins(params, tx, mtm)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		return nil
	}

	primaryField := primaryFields[0]

	// record should be a *SomeModel, we're effectively doing (*record).<pkColumn>
	getKey := func(record reflect.Value) interface{} {
		f := record.Elem().FieldByName(primaryField.Name)
		if !f.IsValid() {
			return nil
		}
		return f.Interface()
	}

	// collect primary key values for all of input
	var keys []interface{}
	for i := 0; i < fresh.Len(); i++ {
		record := fresh.Index(i)
		keys = append(keys, getKey(record))
	}

	cacheAddr, err := c.pagedByKeys(tx, primaryField.DBName, keys, fresh.Type())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cache := cacheAddr.Elem()

	// index cached items by their primary key
	// so we can look them up in O(1) when comparing
	cacheByPK := make(map[interface{}]reflect.Value)
	for i := 0; i < cache.Len(); i++ {
		record := cache.Index(i)
		cacheByPK[getKey(record)] = record
	}

	// compare cached records with fresh records
	var inserts []reflect.Value
	var updates = make(map[interface{}]ChangedFields)

	doneKeys := make(map[interface{}]bool)
	for i := 0; i < fresh.Len(); i++ {
		frec := fresh.Index(i)
		key := getKey(frec)
		if _, ok := doneKeys[key]; ok {
			continue
		}
		doneKeys[key] = true

		if crec, ok := cacheByPK[key]; ok {
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
				updates[key] = cf
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

	for key, rec := range updates {
		err := tx.Table(scope.TableName()).
			Where(
				fmt.Sprintf(
					`"%s" = ?`,
					primaryField.DBName,
				),
				key,
			).
			Updates(rec).
			Error
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	return nil
}
