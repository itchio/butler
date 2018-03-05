package hades

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

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
	Name         string
	Type         reflect.Type
	Children     []*RecordInfo
	Relationship *gorm.Relationship
	ManyToMany   *ManyToMany
	ModelStruct  *gorm.ModelStruct
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

type VisitMap map[*gorm.ModelStruct]bool

func (vm VisitMap) CopyAndMark(ms *gorm.ModelStruct) VisitMap {
	vv := make(VisitMap)
	for k, v := range vm {
		vv[k] = v
	}
	vv[ms] = true
	return vv
}

type RecordInfoMap map[reflect.Type]*RecordInfo

func (c *Context) WalkType(riMap RecordInfoMap, name string, atyp reflect.Type, visited VisitMap, assocs []string) (*RecordInfo, error) {
	consumer := c.Consumer

	if atyp.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("WalkType expects a *Model type, got %v", atyp)
	}
	if atyp.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("WalkType expects a *Model type, got %v", atyp)
	}

	scope := c.ScopeMap[atyp]
	if scope == nil {
		return nil, fmt.Errorf("WalkType expects a *Model but %v is not a registered model type", atyp)
	}
	ms := scope.GetModelStruct()

	if visited[ms] {
		consumer.Debugf("Already visited %v, not recursing.", ms.ModelType)
		return nil, nil
	}
	visited = visited.CopyAndMark(ms)

	consumer.Debugf("Walking type %s: %v, assocs = %v", name, ms.ModelType, assocs)

	ri := &RecordInfo{
		Type:        atyp,
		Name:        name,
		ModelStruct: ms,
	}

	visitField := func(sf *gorm.StructField, explicit bool) error {
		if sf.Relationship == nil {
			if explicit {
				return fmt.Errorf("%s.%s does not describe a relationship", ms.ModelType.Name(), sf.Name)
			}
			return nil
		}

		fieldTyp := sf.Struct.Type
		if fieldTyp.Kind() == reflect.Slice {
			fieldTyp = fieldTyp.Elem()
		}
		if fieldTyp.Kind() != reflect.Ptr {
			return fmt.Errorf("visitField expects a Slice of Ptr, or a Ptr, but got %v", sf.Struct.Type)
		}

		if _, ok := c.ScopeMap[fieldTyp]; !ok {
			if explicit {
				return fmt.Errorf("%s.%s is not an explicitly listed model (%v)", ms.ModelType.Name(), sf.Name, fieldTyp)
			}
			return nil
		}

		child, err := c.WalkType(riMap, sf.Name, fieldTyp, visited, nil)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		if child == nil {
			return nil
		}

		child.Relationship = sf.Relationship

		ri.Children = append(ri.Children, child)
		return nil
	}

	if len(assocs) > 0 {
		sfByName := make(map[string]*gorm.StructField)
		for _, sf := range ms.StructFields {
			sfByName[sf.Name] = sf
		}

		// visit specified fields
		for _, fieldName := range assocs {
			sf, ok := sfByName[fieldName]
			if !ok {
				return nil, fmt.Errorf("No field '%s' in %s", fieldName, atyp)
			}
			err := visitField(sf, true)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		}
	} else {
		// visit all fields
		for _, sf := range ms.StructFields {
			err := visitField(sf, false)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
		}
	}

	riMap[atyp] = ri
	return ri, nil
}
