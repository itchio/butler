package hades

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-errors/errors"
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
}

func (c *Context) NewManyToMany(JoinTable string, L reflect.Type, R reflect.Type) (*ManyToMany, error) {
	var scope *gorm.Scope
	for _, s := range c.ScopeMap {
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

type VisitMap map[reflect.Type]bool

func (vm VisitMap) CopyAndMark(t reflect.Type) VisitMap {
	vv := make(VisitMap)
	for k, v := range vm {
		vv[k] = v
	}
	vv[t] = true
	return vv
}

type RecordInfoMap map[reflect.Type]*RecordInfo

func (c *Context) WalkType(riMap RecordInfoMap, name string, atyp reflect.Type, visited VisitMap, assocs []string) (*RecordInfo, error) {
	consumer := c.Consumer

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
		Scope: c.ScopeMap[atyp],
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

		if _, ok := c.ScopeMap[fieldTyp]; !ok {
			if explicit {
				return fmt.Errorf("Type of assoc '%s' (%v) is not a model", fieldName, fieldTyp)
			}
			return nil
		}

		child, err := c.WalkType(riMap, f.Name, fieldTyp, visited, nil)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if child == nil {
			return nil
		}

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
				mtm, err := c.NewManyToMany(many2manyTable, atyp, elTyp)
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
		return nil
	}

	if len(assocs) > 0 {
		// visit specified fields
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
		// visit all fields
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
