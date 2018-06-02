package hades

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

type JoinRec struct {
	DestinKey interface{}
	Record    reflect.Value
}

type ManyToMany struct {
	Scope *Scope

	JoinTable string

	SourceName        string
	SourceAssocName   string
	SourceDBName      string
	SourceAssocDBName string

	DestinName        string
	DestinAssocName   string
	DestinDBName      string
	DestinAssocDBName string

	// SourceKey => []JoinRec{DestinKey, Record}
	Values map[interface{}][]JoinRec
}

func (c *Context) NewManyToMany(JoinTable string, SourceForeignKeys, DestinationForeignKeys []JoinTableForeignKey) (*ManyToMany, error) {
	scope := c.ScopeMap.ByDBName(JoinTable)
	if scope == nil {
		return nil, errors.Errorf("Could not find model struct for %s: list it explicitly in Models", JoinTable)
	}

	if len(SourceForeignKeys) != 1 {
		return nil, errors.Errorf("For join table %s, expected 1 source foreign keys but got %d",
			JoinTable, len(SourceForeignKeys))
	}
	if len(DestinationForeignKeys) != 1 {
		return nil, errors.Errorf("For join table %s, expected 1 destination foreign keys but got %d",
			JoinTable, len(DestinationForeignKeys))
	}

	sfk := SourceForeignKeys[0]
	dfk := DestinationForeignKeys[0]

	mtm := &ManyToMany{
		JoinTable: JoinTable,
		Scope:     scope,

		SourceName:        FromDBName(sfk.DBName),
		SourceAssocName:   FromDBName(sfk.AssociationDBName),
		SourceDBName:      sfk.DBName,
		SourceAssocDBName: sfk.AssociationDBName,

		DestinName:        FromDBName(dfk.DBName),
		DestinAssocName:   FromDBName(dfk.AssociationDBName),
		DestinDBName:      dfk.DBName,
		DestinAssocDBName: dfk.AssociationDBName,

		Values: make(map[interface{}][]JoinRec),
	}
	return mtm, nil
}

func (mtm *ManyToMany) Mark(Source reflect.Value) {
	sourceKey := Source.Elem().FieldByName(mtm.SourceAssocName).Interface()
	mtm.Values[sourceKey] = make([]JoinRec, 0)
}

func (mtm *ManyToMany) Add(Source reflect.Value, Destin reflect.Value) {
	sourceKey := Source.Elem().FieldByName(mtm.SourceAssocName).Interface()
	destinKey := Destin.Elem().FieldByName(mtm.DestinAssocName).Interface()
	mtm.Values[sourceKey] = append(mtm.Values[sourceKey], JoinRec{
		DestinKey: destinKey,
	})
}

func (mtm *ManyToMany) AddKeys(sourceKey interface{}, destinKey interface{}, record reflect.Value) {
	mtm.Values[sourceKey] = append(mtm.Values[sourceKey], JoinRec{
		DestinKey: destinKey,
		Record:    record,
	})
}

func (mtm *ManyToMany) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("JoinTable: %s", mtm.JoinTable))
	lines = append(lines, fmt.Sprintf("SourceForeignKey: %s / %s", mtm.SourceName, mtm.SourceAssocName))
	lines = append(lines, fmt.Sprintf("DestinForeignKey: %s / %s", mtm.DestinName, mtm.DestinAssocName))
	for sourceKey, destinKeys := range mtm.Values {
		lines = append(lines, fmt.Sprintf("SourceKey %v", sourceKey))
		for _, destinKey := range destinKeys {
			lines = append(lines, fmt.Sprintf("  - DestinKey %v", destinKey))
		}
	}
	return strings.Join(lines, "\n")
}

type RecordInfo struct {
	Field        AssocField
	Type         reflect.Type
	Children     []*RecordInfo
	Relationship *Relationship
	ManyToMany   *ManyToMany
	ModelStruct  *ModelStruct
}

func (ri *RecordInfo) Name() string {
	return ri.Field.Name()
}

func (ri *RecordInfo) String() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("- %s: %s", ri.Name(), ri.Type.String()))
	for _, c := range ri.Children {
		for _, cl := range strings.Split(c.String(), "\n") {
			lines = append(lines, "  "+cl)
		}
	}
	return strings.Join(lines, "\n")
}

type RecordInfoMap map[reflect.Type]*RecordInfo

func (c *Context) WalkType(riMap RecordInfoMap, field AssocField, atyp reflect.Type) (*RecordInfo, error) {
	if atyp.Kind() != reflect.Ptr {
		return nil, errors.Errorf("WalkType expects a *Model type, got %v", atyp)
	}
	if atyp.Elem().Kind() != reflect.Struct {
		return nil, errors.Errorf("WalkType expects a *Model type, got %v", atyp)
	}

	scope := c.ScopeMap.ByType(atyp)
	if scope == nil {
		return nil, errors.Errorf("WalkType expects a *Model but %v is not a registered model type", atyp)
	}
	ms := scope.GetModelStruct()

	ri := &RecordInfo{
		Type:        atyp,
		Field:       field,
		ModelStruct: ms,
	}

	// visit specified assocs
	for _, assoc := range field.Children() {
		sf, ok := ms.StructFieldsByName[assoc.Name()]
		if !ok {
			return nil, errors.Errorf("No field '%s' in %s", assoc.Name(), atyp)
		}
		if sf.Relationship == nil {
			return nil, errors.Errorf("%s.%s does not describe a relationship", ms.ModelType.Name(), sf.Name)
		}

		fieldTyp := sf.Struct.Type
		if fieldTyp.Kind() == reflect.Slice {
			fieldTyp = fieldTyp.Elem()
		}
		if fieldTyp.Kind() != reflect.Ptr {
			return nil, errors.Errorf("visitField expects a Slice of Ptr, or a Ptr, but got %v", sf.Struct.Type)
		}

		if c.ScopeMap.ByType(fieldTyp) == nil {
			return nil, errors.Errorf("%s.%s is not an explicitly listed model (%v)", ms.ModelType.Name(), sf.Name, fieldTyp)
		}

		child, err := c.WalkType(riMap, assoc, fieldTyp)
		if err != nil {
			return nil, errors.WithMessage(err, "walking type of child")
		}

		if child == nil {
			return nil, nil
		}

		child.Relationship = sf.Relationship

		if sf.Relationship.Kind == "many_to_many" {
			jth := sf.Relationship.JoinTableHandler
			djth, ok := jth.(*JoinTableHandler)
			if !ok {
				return nil, errors.Errorf("Expected sf.Relationship.JoinTableHandler to be the default JoinTableHandler type, but it's %v", reflect.TypeOf(jth))
			}

			mtm, err := c.NewManyToMany(djth.TableName, jth.SourceForeignKeys(), jth.DestinationForeignKeys())
			if err != nil {
				return nil, errors.WithMessage(err, "creating ManyToMany relation")
			}
			child.ManyToMany = mtm
		}

		ri.Children = append(ri.Children, child)
	}

	riMap[atyp] = ri
	return ri, nil
}
