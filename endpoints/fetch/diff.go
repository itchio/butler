package fetch

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/database"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
)

type RecordInfo struct {
	Name     string
	Type     reflect.Type
	Children []*RecordInfo
}

func (ri *RecordInfo) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("- %s: %s", ri.Name, ri.Type.String()))
	for _, c := range ri.Children {
		for _, cl := range strings.Split(c.String(), "\n") {
			lines = append(lines, "  "+cl)
		}
	}
	return strings.Join(lines, "\n")
}

type AllEntities map[reflect.Type]EntityMap
type EntityMap []interface{}

func SaveRecursive(db *gorm.DB, consumer *state.Consumer, rec interface{}, assocs []string) error {
	startTime := time.Now()

	tx := db.Begin()
	success := false

	database.SetLogger(tx, consumer)

	defer func() {
		if success {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	val := reflect.ValueOf(rec)

	modelTyps := make(map[reflect.Type]bool)
	for _, m := range database.Models {
		mtyp := reflect.TypeOf(m)
		modelTyps[mtyp] = true
	}

	var walkType func(name string, atyp reflect.Type, assocs []string) (*RecordInfo, error)
	walkType = func(name string, atyp reflect.Type, assocs []string) (*RecordInfo, error) {
		consumer.Debugf("walking type %s: %v, assocs = %v", name, atyp, assocs)
		if atyp.Kind() == reflect.Slice {
			atyp = atyp.Elem()
		}

		ri := &RecordInfo{
			Type: atyp,
			Name: name,
		}

		if atyp.Kind() == reflect.Ptr {
			atyp = atyp.Elem()
		}

		if atyp.Kind() != reflect.Struct {
			return nil, nil
		}

		visitField := func(f reflect.StructField, explicit bool) error {
			fieldTyp := f.Type
			fieldName := f.Name

			if fieldTyp.Kind() == reflect.Slice {
				fieldTyp = fieldTyp.Elem()
			}

			if _, ok := modelTyps[fieldTyp]; ok {
				child, err := walkType(f.Name, fieldTyp, nil)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				if child != nil {
					ri.Children = append(ri.Children, child)
				}
			} else {
				if explicit {
					return fmt.Errorf("Type of assoc '%s' (%v) is not a model", fieldName, fieldTyp)
				}
			}
			return nil
		}

		if assocs != nil {
			for _, fieldName := range assocs {
				consumer.Debugf("looking at assoc %s", fieldName)
				f, ok := atyp.FieldByName(fieldName)
				if !ok {
					return nil, fmt.Errorf("No field '%s' in %s", fieldName, atyp)
				}
				err := visitField(f, true)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}
			}
		} else {
			for i := 0; i < atyp.NumField(); i++ {
				f := atyp.Field(i)
				err := visitField(f, false)
				if err != nil {
					return nil, errors.Wrap(err, 0)
				}
			}
		}
		return ri, nil
	}
	tree, err := walkType("<root>", val.Type(), assocs)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Debugf("record tree:\n%s", tree)

	entities := make(AllEntities)
	addEntity := func(v reflect.Value) error {
		typ := v.Type()
		if _, ok := modelTyps[typ]; !ok {
			return fmt.Errorf("not a model type: %s", typ)
		}
		entities[typ] = append(entities[typ], v.Interface())
		return nil
	}

	var walk func(v reflect.Value, ri *RecordInfo, persist bool) error

	var numVisited int64
	visit := func(v reflect.Value, ri *RecordInfo, persist bool) error {
		if persist {
			numVisited++
			err := addEntity(v)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		if v.Kind() == reflect.Ptr {
			v = reflect.Indirect(v)
		}
		if v.Kind() != reflect.Struct {
			return fmt.Errorf("expected a struct, but stuck with %v", v)
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
			err := walk(field, child, true)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	walk = func(v reflect.Value, ri *RecordInfo, persist bool) error {
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				err := visit(v.Index(i), ri, persist)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			}
		} else {
			err := visit(v, ri, persist)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	persistRoot := assocs == nil
	err = walk(val, tree, persistRoot)
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

	stats := &SaveManyStats{}
	for _, m := range entities {
		err := SaveMany(tx, consumer, m, stats)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	consumer.Infof("Inserted %d, Updated %d, Current %d in %s",
		stats.Inserted,
		stats.Updated,
		stats.Current,
		time.Since(startTime),
	)

	success = true
	return nil
}

type SaveManyStats struct {
	Inserted int64
	Updated  int64
	Current  int64
}

func SaveMany(db *gorm.DB, consumer *state.Consumer, inputIface interface{}, stats *SaveManyStats) error {
	db = db.Set("gorm:save_associations", false)

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
	scope := db.NewScope(first.Interface())
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
		return fmt.Errorf("Have %d primary keys for %s, don't know what to do", len(pkFields), modelName)
	}

	pkField := pkFields[0]

	// record should be a *SomeModel, we're effectively doing (*record).<pkColumn>
	getPk := func(record reflect.Value) interface{} {
		f := reflect.Indirect(record).FieldByName(pkField.Name)
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
	cacheAddr := reflect.New(fresh.Type())
	err = db.Where(fmt.Sprintf("%s in (?)", pkField.DBName), pks).Find(cacheAddr.Interface()).Error
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
			ifrec := reflect.Indirect(frec).Interface()
			icrec := reflect.Indirect(crec).Interface()

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

	stats.Inserted += int64(len(inserts))
	stats.Updated += int64(len(updates))
	stats.Current += int64(fresh.Len() - len(updates) - len(inserts))

	if len(inserts) > 0 {
		for _, rec := range inserts {
			err := db.Create(rec.Interface()).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	if len(updates) > 0 {
		for pk, rec := range updates {
			err := db.Table(scope.TableName()).Where(fmt.Sprintf("%s = ?", pkField.DBName), pk).Updates(rec).Error
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
	}

	return nil
}
