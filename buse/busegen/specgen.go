package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-errors/errors"
)

type Spec struct {
	Requests      []*RequestSpec      `json:"requests"`
	Notifications []*NotificationSpec `json:"notifications"`
	StructTypes   []*StructTypeSpec   `json:"types"`
	EnumTypes     []*EnumTypeSpec     `json:"types"`
	VersionNote   string              `json:"versionNote"`
}

type RequestSpec struct {
	Method string      `json:"method"`
	Doc    string      `json:"doc"`
	Caller string      `json:"caller"`
	Params *StructSpec `json:"params"`
	Result *StructSpec `json:"result"`
}

type StructTypeSpec struct {
	Name   string       `json:"name"`
	Doc    string       `json:"doc"`
	Fields []*FieldSpec `json:"fields"`
}

type EnumTypeSpec struct {
	Name   string           `json:"name"`
	Doc    string           `json:"doc"`
	Values []*EnumValueSpec `json:"values"`
}

type EnumValueSpec struct {
	Name  string `json:"name"`
	Doc   string `json:"doc"`
	Value string `json:"value"`
}

type StructSpec struct {
	Fields []*FieldSpec `json:"fields"`
}

type FieldSpec struct {
	Name string `json:"name"`
	Doc  string `json:"doc"`
	Type string `json:"type"`
}

type NotificationSpec struct {
	Method string      `json:"method"`
	Doc    string      `json:"doc"`
	Params *StructSpec `json:"params"`
}

func (bc *BuseContext) GenerateSpec() error {
	bc.Task("Generating JSON spec")

	doc := bc.NewBusegenRelativeDoc("spec/buse.json")

	rev := bc.Revision()
	versionNote := fmt.Sprintf("Generated on %s against butler@%s", bc.Timestamp(), rev)
	s := &Spec{
		VersionNote: versionNote,
	}

	scope := newScope(bc)
	must(scope.Assimilate("github.com/itchio/butler/buse", "types.go"))
	must(scope.Assimilate("github.com/itchio/go-itchio", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/configurator", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/installer/bfs", "receipt.go"))

	encodeStruct := func(entry *Entry) []*FieldSpec {
		var res []*FieldSpec
		for _, sf := range entry.structFields {
			fs := &FieldSpec{
				Name: sf.name,
				Type: sf.typeString,
				Doc:  strings.Join(sf.doc, "\n"),
			}
			res = append(res, fs)
		}
		return res
	}

	for _, category := range scope.categoryList {
		cat := scope.categories[category]
		for _, entry := range cat.entries {
			switch entry.kind {
			case EntryKindParams:
				params := entry
				result := scope.FindEntry(strings.TrimSuffix(entry.typeName, "Params") + "Result")
				var caller string
				switch entry.caller {
				case CallerClient:
					caller = "client"
				case CallerServer:
					caller = "server"
				}

				rs := &RequestSpec{
					Method: entry.name,
					Caller: caller,
					Params: &StructSpec{
						Fields: encodeStruct(params),
					},
					Result: &StructSpec{
						Fields: encodeStruct(result),
					},
					Doc: strings.Join(params.doc, "\n"),
				}
				s.Requests = append(s.Requests, rs)
			case EntryKindNotification:
				ns := &NotificationSpec{
					Method: entry.name,
					Params: &StructSpec{
						Fields: encodeStruct(entry),
					},
					Doc: strings.Join(entry.doc, "\n"),
				}
				s.Notifications = append(s.Notifications, ns)
			}
		}
	}

	js, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	doc.Line(string(js))
	doc.Commit("")
	doc.Write()

	return nil
}
