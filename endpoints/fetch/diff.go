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
	input := reflect.ValueOf(inputIface)
	if input.Kind() != reflect.Slice {
		return errors.New("diff needs a slice")
	}

	if input.Len() == 0 {
		consumer.Infof("Nothing to persist (0 input records)")
		return nil
	}

	first := input.Index(0).Elem()
	fresh := reflect.MakeSlice(reflect.SliceOf(first.Type()), input.Len(), input.Len())
	for i := 0; i < input.Len(); i++ {
		record := input.Index(i).Elem()
		fresh.Index(i).Set(record)
	}

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

	if len(pkColumns) != 1 {
		return fmt.Errorf("Have %d primary keys, don't know what to do", len(pkColumns))
	}

	pkColumn := pkColumns[0]

	getPk := func(record reflect.Value) interface{} {
		return reflect.Indirect(record).FieldByName(pkColumn).Interface()
	}

	var pks []interface{}
	for i := 0; i < fresh.Len(); i++ {
		record := fresh.Index(i)
		pks = append(pks, getPk(record))
	}

	var err error

	cacheAddr := reflect.New(fresh.Type())
	err = tx.Where(fmt.Sprintf("%s in (?)", pkColumn), pks).Find(cacheAddr.Interface()).Error
	if err != nil {
		return errors.Wrap(err, 0)
	}

	cache := reflect.Indirect(cacheAddr)

	cacheByPK := make(map[interface{}]reflect.Value)
	for i := 0; i < cache.Len(); i++ {
		record := cache.Index(i)
		cacheByPK[getPk(record)] = record
	}

	var insertPKs []interface{}
	var updatePKs []interface{}

	for i := 0; i < fresh.Len(); i++ {
		frec := fresh.Index(i)
		pk := getPk(frec)
		if crec, ok := cacheByPK[pk]; ok {
			ifrec := reflect.Indirect(frec).Interface()
			icrec := reflect.Indirect(crec).Interface()
			if !RecordEqual(ifrec, icrec) {
				updatePKs = append(updatePKs, pk)
			}
		} else {
			insertPKs = append(insertPKs, pk)
		}
	}

	consumer.Statf("%d records to insert", len(insertPKs))
	consumer.Statf("%d records to update", len(updatePKs))
	consumer.Statf("%d records valid in cache", fresh.Len()-len(updatePKs)-len(insertPKs))

	// persistAddr := reflect.New(reflect.TypeOf(freshIface))

	return nil
}
