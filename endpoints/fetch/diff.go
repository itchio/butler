package fetch

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
)

func diff(tx *gorm.DB, consumer *state.Consumer, inputIface interface{}) error {
	tx = tx.Set("gorm:save_associations", false)

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
	var pkColumns []string
	fs := scope.Fields()
	for _, f := range fs {
		if f.IsPrimaryKey {
			pkColumns = append(pkColumns, f.Name)
		}
	}

	consumer.Infof("Persisting %d records for %s, primary keys: (%s)", fresh.Len(), modelName, strings.Join(pkColumns, ", "))

	// this will happen for associations, we should have another codepath for that
	if len(pkColumns) != 1 {
		return fmt.Errorf("Have %d primary keys, don't know what to do", len(pkColumns))
	}

	pkColumn := pkColumns[0]

	// record should be a *SomeModel, we're effecctively doing (*record).<pkColumn>
	getPk := func(record reflect.Value) interface{} {
		return reflect.Indirect(record).FieldByName(pkColumn).Interface()
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
	cacheAddr := reflect.New(fresh.Type())
	err = tx.Where(fmt.Sprintf("%s in (?)", pkColumn), pks).Find(cacheAddr.Interface()).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cache := reflect.Indirect(cacheAddr)

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

	for i := 0; i < fresh.Len(); i++ {
		frec := fresh.Index(i)
		pk := getPk(frec)
		if crec, ok := cacheByPK[pk]; ok {
			// frec and crec are *SomeModel, but `RecordEqual` ignores pointer
			// equality - we want to compare the contents of the struct
			// so we indirect to SomeModel here.
			ifrec := reflect.Indirect(frec).Interface()
			icrec := reflect.Indirect(crec).Interface()

			cf, err := DiffRecord(ifrec, icrec)
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

	consumer.Statf("%d records to insert", len(inserts))
	consumer.Statf("%d records to update", len(updates))
	consumer.Statf("%d records valid in cache", fresh.Len()-len(updates)-len(inserts))

	if len(inserts) > 0 {
		for _, rec := range inserts {
			err := tx.Debug().Create(rec.Interface()).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	if len(updates) > 0 {
		for pk, rec := range updates {
			err := tx.Debug().Table(scope.TableName()).Where(fmt.Sprintf("%s = ?", pkColumn), pk).Updates(rec).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	return nil
}
