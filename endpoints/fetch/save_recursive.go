package fetch

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/database"
	"github.com/itchio/wharf/state"
	"github.com/jinzhu/gorm"
)

type Assoc int

const (
	AssocNone = iota
	AssocBelongsTo
	AssocHasOne
	AssocHasMany
	AssocManyToMany
)

func (a Assoc) String() string {
	switch a {
	case AssocNone:
		return "AssocNone"
	case AssocBelongsTo:
		return "AssocBelongsTo"
	case AssocHasOne:
		return "AssocHasOne"
	case AssocHasMany:
		return "AssocHasMany"
	case AssocManyToMany:
		return "AssocManyToMany"
	default:
		return "<invalid assoc value>"
	}
}

type JoinRec struct {
	RPK    interface{}
	Record reflect.Value
}

type ManyToMany struct {
	JoinTable string
	Scope     *gorm.Scope

	LPKColumn string
	RPKColumn string

	// LPK => []RPK
	Values map[interface{}][]JoinRec

	initialized bool
}

type ScopeMap map[reflect.Type]*gorm.Scope

func NewManyToMany(ScopeMap ScopeMap, JoinTable string, L reflect.Type, R reflect.Type) (*ManyToMany, error) {
	var scope *gorm.Scope
	for _, s := range ScopeMap {
		if s.TableName() == JoinTable {
			scope = s
		}
	}

	if scope == nil {
		return nil, fmt.Errorf("Could not find model struct for %s: list it explicitly in Models", JoinTable)
	}

	mtm := &ManyToMany{
		JoinTable: JoinTable,
		Scope:     scope,
		// TODO: handle different FKs
		LPKColumn: L.Name() + "ID",
		RPKColumn: R.Name() + "ID",
		Values:    make(map[interface{}][]JoinRec),
	}
	return mtm, nil
}

func (mtm *ManyToMany) Add(L reflect.Value, R reflect.Value) {
	// TODO: handle different PKs
	lpk := L.Elem().FieldByName("ID").Interface()
	rpk := R.Elem().FieldByName("ID").Interface()
	mtm.Values[lpk] = append(mtm.Values[lpk], JoinRec{
		RPK: rpk,
	})
}

func (mtm *ManyToMany) AddPKs(lpk interface{}, rpk interface{}, record reflect.Value) {
	mtm.Values[lpk] = append(mtm.Values[lpk], JoinRec{
		RPK:    rpk,
		Record: record,
	})
}

func (mtm *ManyToMany) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("JoinTable: %s", mtm.JoinTable))
	lines = append(lines, fmt.Sprintf("LPKColumn: %s", mtm.LPKColumn))
	lines = append(lines, fmt.Sprintf("RPKColumn: %s", mtm.RPKColumn))
	for lpk, rpks := range mtm.Values {
		lines = append(lines, fmt.Sprintf("LPK %v", lpk))
		for _, rpk := range rpks {
			lines = append(lines, fmt.Sprintf("  - RPK %v", rpk))
		}
	}
	return strings.Join(lines, "\n")
}

type RecordInfo struct {
	Name       string
	Type       reflect.Type
	Children   []*RecordInfo
	Assoc      Assoc
	ManyToMany *ManyToMany
	ForeignKey string
	Scope      *gorm.Scope
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

type SaveParams struct {
	// Record to save
	Record interface{}

	// Fields to save instead of the top-level record
	Assocs []string

	// Disable deleting join table entries (useful for partial data)
	PartialJoins []string
}

type VisitMap map[reflect.Type]bool

func (vm VisitMap) CopyAndMark(t reflect.Type) VisitMap {
	vv := make(VisitMap)
	for k, v := range vm {
		vv[k] = v
	}
	vv[t] = true
	return vv
}

func SaveRecursive(db *gorm.DB, consumer *state.Consumer, params *SaveParams) error {
	if params == nil {
		return errors.New("SaveRecursive: params cannot be nil")
	}
	rec := params.Record
	assocs := params.Assocs

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

	scopeMap := make(ScopeMap)
	for _, m := range database.Models {
		mtyp := reflect.TypeOf(m)
		scopeMap[mtyp] = db.NewScope(m)
	}

	riMap := make(map[reflect.Type]*RecordInfo)

	var walkType func(name string, atyp reflect.Type, visited VisitMap, assocs []string) (*RecordInfo, error)
	walkType = func(name string, atyp reflect.Type, visited VisitMap, assocs []string) (*RecordInfo, error) {
		if visited[atyp] {
			consumer.Debugf("Already visited %v, not recursing.", atyp)
			return nil, nil
		}
		visited = visited.CopyAndMark(atyp)

		consumer.Debugf("walking type %s: %v, assocs = %v", name, atyp, assocs)
		if atyp.Kind() == reflect.Slice {
			atyp = atyp.Elem()
		}
		refAtyp := atyp

		ri := &RecordInfo{
			Type:  atyp,
			Name:  name,
			Scope: scopeMap[atyp],
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
			wasSlice := false

			if fieldTyp.Kind() == reflect.Slice {
				wasSlice = true
				fieldTyp = fieldTyp.Elem()
			}

			if _, ok := scopeMap[fieldTyp]; ok {
				child, err := walkType(f.Name, fieldTyp, visited, nil)
				if err != nil {
					return errors.Wrap(err, 0)
				}
				if child != nil {
					elTyp := fieldTyp.Elem()
					gormtag := f.Tag.Get("gorm")
					tokens := strings.Split(gormtag, ";")
					var many2manyTable = ""
					var foreignKey = ""
					for _, t := range tokens {
						token := strings.ToLower(strings.TrimSpace(t))
						if strings.HasPrefix(token, "many2many:") {
							many2manyTable = strings.TrimPrefix(token, "many2many:")
						} else if strings.HasPrefix(token, "foreignkey:") {
							foreignKey = strings.TrimPrefix(token, "foreignkey:")
						}
					}

					if wasSlice {
						if foreignKey == "" {
							foreignKey = gorm.ToDBName(atyp.Name() + "ID")
						}

						if many2manyTable != "" {
							consumer.Infof("%s <many to many> %s (join table %s)", atyp.Name(), elTyp.Name(), many2manyTable)
							child.Assoc = AssocManyToMany
							mtm, err := NewManyToMany(scopeMap, many2manyTable, atyp, elTyp)
							if err != nil {
								return errors.Wrap(err, 0)
							}
							child.ManyToMany = mtm
						} else {
							consumer.Infof("%s <has many> %s (via %s)", atyp.Name(), elTyp.Name(), foreignKey)
							child.Assoc = AssocHasMany
						}
					} else if _, ok := atyp.FieldByName(fieldName + "ID"); ok {
						if foreignKey == "" {
							foreignKey = gorm.ToDBName(fieldName + "ID")
						}

						consumer.Infof("%s <belongs to> %s (via %s)", atyp.Name(), elTyp.Name(), foreignKey)
						child.Assoc = AssocBelongsTo
					} else if _, ok := elTyp.FieldByName(atyp.Name() + "ID"); ok {
						if foreignKey == "" {
							foreignKey = gorm.ToDBName(atyp.Name() + "ID")
						}

						consumer.Infof("%s <has one> %s (via %s)", atyp.Name(), elTyp.Name(), foreignKey)
						child.Assoc = AssocHasOne
					}

					if child.Assoc != AssocNone {
						var fktyp reflect.Type
						switch child.Assoc {
						case AssocHasOne, AssocHasMany:
							fktyp = elTyp
						case AssocBelongsTo:
							fktyp = atyp
						}

						if fktyp != nil {
							foundFK := false
							for i := 0; i < fktyp.NumField(); i++ {
								ff := fktyp.Field(i)
								if gorm.ToDBName(ff.Name) == foreignKey {
									child.ForeignKey = ff.Name
									foundFK = true
									break
								}
							}

							if !foundFK {
								return fmt.Errorf("For %v, didn't find field for foreign key %s", fktyp, foreignKey)
							}
						}
					}

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

		riMap[refAtyp] = ri
		return ri, nil
	}
	tree, err := walkType("<root>", val.Type(), make(VisitMap), assocs)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Debugf("record tree:\n%s", tree)

	entities := make(AllEntities)
	addEntity := func(v reflect.Value) error {
		typ := v.Type()
		if _, ok := scopeMap[typ]; !ok {
			return fmt.Errorf("not a model type: %s", typ)
		}
		entities[typ] = append(entities[typ], v.Interface())
		return nil
	}

	var walk func(parent reflect.Value, v reflect.Value, ri *RecordInfo, persist bool) error

	var numVisited int64
	visit := func(parent reflect.Value, v reflect.Value, ri *RecordInfo, persist bool) error {
		ov := v

		if persist {
			if ri.Assoc != AssocNone {
				p := parent
				switch ri.Assoc {
				case AssocHasMany, AssocHasOne:
					pkField := p.Elem().FieldByName("ID")
					fkField := v.Elem().FieldByName(ri.ForeignKey)
					fkField.Set(pkField)
				case AssocBelongsTo:
					pkField := v.Elem().FieldByName("ID")
					fkField := p.Elem().FieldByName(ri.ForeignKey)
					fkField.Set(pkField)
				case AssocManyToMany:
					ri.ManyToMany.Add(p, v)
				}
			}

			numVisited++
			err := addEntity(v)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}

		if v.Kind() != reflect.Ptr {
			return fmt.Errorf("expected a pointer, but got with %v", v)
		}
		v = v.Elem()

		if v.Kind() != reflect.Struct {
			return fmt.Errorf("expected a struct, but got with %v", v)
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
			err := walk(ov, field, child, true)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	walk = func(parent reflect.Value, v reflect.Value, ri *RecordInfo, persist bool) error {
		if v.Kind() == reflect.Slice {
			for i := 0; i < v.Len(); i++ {
				err := visit(parent, v.Index(i), ri, persist)
				if err != nil {
					return errors.Wrap(err, 0)
				}
			}
		} else {
			err := visit(parent, v, ri, persist)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}

	persistRoot := assocs == nil
	err = walk(reflect.Zero(reflect.TypeOf(0)), val, tree, persistRoot)
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
		err := SaveMany(params, tx, consumer, scopeMap, m, stats)
		if err != nil {
			return errors.Wrap(err, 0)
		}
	}

	for _, ri := range riMap {
		if ri.ManyToMany != nil {
			err := SaveJoins(params, tx, consumer, ri.ManyToMany)
			if err != nil {
				return errors.Wrap(err, 0)
			}
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

func SaveMany(params *SaveParams, tx *gorm.DB, consumer *state.Consumer, scopeMap ScopeMap, inputIface interface{}, stats *SaveManyStats) error {
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

		lpkName := strings.TrimSuffix(bestLPK.Name, "ID")
		rpkName := strings.TrimSuffix(bestRPK.Name, "ID")
		var L, R reflect.Type
		for _, pkf := range scope.Fields() {
			if pkf.Name == lpkName {
				L = pkf.Struct.Type.Elem()
			}
			if pkf.Name == rpkName {
				R = pkf.Struct.Type.Elem()
			}
		}

		if L == nil {
			return fmt.Errorf("Internal error: could not find LPK %s in %s", lpkName, scope.TableName())
		}
		if R == nil {
			return fmt.Errorf("Internal error: could not find RPK %s in %s", rpkName, scope.TableName())
		}

		mtm, err := NewManyToMany(scopeMap, scope.TableName(), L, R)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		for lpk, recs := range valueMap {
			for _, rec := range recs {
				rpk := rec.Elem().FieldByName(bestRPK.Name).Interface()
				mtm.AddPKs(lpk, rpk, rec)
			}
		}

		err = SaveJoins(params, tx, consumer, mtm)
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
		// returns a pointer to slice
		result := reflect.New(sliceType)

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
			page := reflect.New(sliceType)
			err = tx.Where(fmt.Sprintf("%s in (?)", pkDBName), remainingItems[:pageSize]).Find(page.Interface()).Error
			if err != nil {
				return result, errors.Wrap(err, 0)
			}

			result = reflect.AppendSlice(result.Elem(), page.Elem())
			remainingItems = remainingItems[pageSize:]
		}

		return result, nil
	}

	cacheAddr, err := pagedByPK(pkField.DBName, pks, fresh.Type())
	if err != nil {
		return errors.Wrap(err, 0)
	}

	// cacheAddr := reflect.New(fresh.Type())
	// err = tx.Where(fmt.Sprintf("%s in (?)", pkField.DBName), pks).Find(cacheAddr.Interface()).Error
	// if err != nil {
	// 	return errors.Wrap(err, 0)
	// }

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

	stats.Inserted += int64(len(inserts))
	stats.Updated += int64(len(updates))
	stats.Current += int64(fresh.Len() - len(updates) - len(inserts))

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

func SaveJoins(params *SaveParams, tx *gorm.DB, consumer *state.Consumer, mtm *ManyToMany) error {
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
