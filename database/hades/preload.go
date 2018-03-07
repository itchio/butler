package hades

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-errors/errors"
	"github.com/jinzhu/gorm"
)

type PreloadParams struct {
	Record interface{}

	// Fields to preload, for example []string{"CollectionGames", "CollectionGames.Game"}
	Fields []PreloadField
}

type PreloadCB func(db *gorm.DB) *gorm.DB

type PreloadField struct {
	Name    string
	OrderBy string
}

type Node struct {
	Name     string
	Field    PreloadField
	Children map[string]*Node
}

func (n *Node) cb(db *gorm.DB) *gorm.DB {
	f := n.Field
	if f.OrderBy != "" {
		db = db.Order(f.OrderBy)
	}
	return db
}

func NewNode(name string) *Node {
	return &Node{
		Name:     name,
		Children: make(map[string]*Node),
	}
}

func (n *Node) String() string {
	var res []string
	var orderByStr string
	if n.Field.OrderBy != "" {
		orderByStr = fmt.Sprintf(" ORDER BY %s", n.Field.OrderBy)
	}
	res = append(res, fmt.Sprintf("- %s%s", n.Name, orderByStr))
	for _, c := range n.Children {
		for _, cl := range strings.Split(c.String(), "\n") {
			res = append(res, "  "+cl)
		}
	}
	return strings.Join(res, "\n")
}

func (n *Node) Add(pf PreloadField) {
	tokens := strings.Split(pf.Name, ".")
	name := tokens[0]

	c, ok := n.Children[name]
	if !ok {
		c = NewNode(name)
		n.Children[name] = c
	}

	if len(tokens) > 1 {
		pfc := pf
		pfc.Name = strings.Join(tokens[1:], ".")
		c.Add(pfc)
	} else {
		c.Field = pf
	}
}

func (c *Context) Preload(db *gorm.DB, params *PreloadParams) error {
	consumer := c.Consumer
	rec := params.Record
	if len(params.Fields) == 0 {
		return errors.New("Preload expects a non-empty list in Fields")
	}

	val := reflect.ValueOf(rec)
	valtyp := val.Type()
	if valtyp.Kind() == reflect.Slice {
		if val.Len() == 0 {
			consumer.Debugf("nothing to preload (0-len slice passed)")
			return nil
		}
		valtyp = valtyp.Elem()
	}
	if valtyp.Kind() != reflect.Ptr {
		return fmt.Errorf("Preload expects a []*Model or *Model, but it was passed a %v instead", val.Type())
	}

	riMap := make(RecordInfoMap)
	typeTree, err := c.WalkType(riMap, "<root>", valtyp, make(VisitMap), nil)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	consumer.Debugf("typeTree:\n%s", typeTree)

	valTree := NewNode("<root>")
	for _, field := range params.Fields {
		valTree.Add(field)
	}

	consumer.Debugf("valTree:\n%s", valTree)

	var walk func(p reflect.Value, pri *RecordInfo, pvt *Node) error
	walk = func(p reflect.Value, pri *RecordInfo, pvt *Node) error {
		for _, cvt := range pvt.Children {
			var cri *RecordInfo
			for _, c := range pri.Children {
				if c.Name == cvt.Name {
					cri = c
					break
				}
			}
			if cri == nil {
				return fmt.Errorf("Relation not found: %s.%s", pri.Name, cvt.Name)
			}

			ptyp := p.Type()
			wasSlice := false

			if ptyp.Kind() == reflect.Slice {
				wasSlice = true
				ptyp = ptyp.Elem()
			}
			if ptyp.Kind() != reflect.Ptr {
				return fmt.Errorf("walk expects a []*Model or *Model, but it was passed a %v instead", p.Type())
			}

			sourceKind := "single"
			if wasSlice {
				sourceKind = "many"
			}

			destKind := ""
			switch cri.Relationship.Kind {
			case "has_many":
				destKind = "many"
			case "has_single", "belongs_to":
				destKind = "single"
			default:
				return fmt.Errorf("Preload doesn't know how to handle %s relationships", cri.Relationship.Kind)
			}

			consumer.Debugf("Preloading %s %s on %s %s", destKind, cri.Name, sourceKind, pvt.Name)
			freshAddr := reflect.New(reflect.SliceOf(cri.Type))

			var ps reflect.Value
			if p.Type().Kind() == reflect.Slice {
				ps = p
			} else {
				ps = reflect.MakeSlice(reflect.SliceOf(p.Type()), 1, 1)
				ps.Index(0).Set(p)
			}

			switch cri.Relationship.Kind {
			case "has_many":
				var keys []interface{}
				for i := 0; i < ps.Len(); i++ {
					keys = append(keys, ps.Index(i).Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface())
				}

				var err error
				freshAddr, err = c.pagedByKeys(db, cri.Relationship.ForeignDBNames[0], keys, reflect.SliceOf(cri.Type), cvt.cb)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				consumer.Debugf("In has_many, found %d values (for ps of len %d)", freshAddr.Elem().Len(), ps.Len())

				pByFK := make(map[interface{}]reflect.Value)
				for i := 0; i < ps.Len(); i++ {
					rec := ps.Index(i)
					fk := rec.Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface()
					pByFK[fk] = rec

					// reset slices so if preload is called more than once,
					// it doesn't keep appending
					field := rec.Elem().FieldByName(cvt.Name)
					field.Set(reflect.New(field.Type()).Elem())
				}

				fresh := freshAddr.Elem()
				for i := 0; i < fresh.Len(); i++ {
					fk := fresh.Index(i).Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface()
					if p, ok := pByFK[fk]; ok {
						dest := p.Elem().FieldByName(cvt.Name)
						dest.Set(reflect.Append(dest, fresh.Index(i)))
					}
				}
			case "has_one", "belongs_to":
				var keys []interface{}
				for i := 0; i < ps.Len(); i++ {
					keys = append(keys, ps.Index(i).Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface())
				}

				var err error
				freshAddr, err = c.pagedByKeys(db, cri.Relationship.AssociationForeignDBNames[0], keys, reflect.SliceOf(cri.Type), cvt.cb)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				fresh := freshAddr.Elem()
				freshByFK := make(map[interface{}]reflect.Value)
				for i := 0; i < fresh.Len(); i++ {
					rec := fresh.Index(i)
					fk := rec.Elem().FieldByName(cri.Relationship.AssociationForeignFieldNames[0]).Interface()
					freshByFK[fk] = rec
				}

				for i := 0; i < ps.Len(); i++ {
					prec := ps.Index(i)
					fk := prec.Elem().FieldByName(cri.Relationship.ForeignFieldNames[0]).Interface()
					if crec, ok := freshByFK[fk]; ok {
						prec.Elem().FieldByName(cvt.Name).Set(crec)
					}
				}
			}

			fresh := freshAddr.Elem()

			err = walk(fresh, cri, cvt)
			if err != nil {
				return errors.Wrap(err, 0)
			}
		}
		return nil
	}
	err = walk(val, typeTree, valTree)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
