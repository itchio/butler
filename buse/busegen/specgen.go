package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-errors/errors"
	"github.com/itchio/butler/buse/busegen/spec"
)

func (bc *BuseContext) GenerateSpec() error {
	bc.Task("Generating JSON spec")

	doc := bc.NewBusegenRelativeDoc("spec/buse.json")

	rev := bc.Revision()
	versionNote := fmt.Sprintf("Generated on %s against butler@%s", bc.Timestamp(), rev)
	s := &spec.Spec{
		VersionNote: versionNote,
	}

	scope := newScope(bc)
	must(scope.Assimilate("github.com/itchio/butler/buse", "types.go"))
	must(scope.Assimilate("github.com/itchio/go-itchio", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/configurator", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/installer/bfs", "receipt.go"))

	encodeStruct := func(entry *Entry) []*spec.FieldSpec {
		var res []*spec.FieldSpec
		for _, sf := range entry.structFields {
			fs := &spec.FieldSpec{
				Name: sf.name,
				Type: sf.typeString,
				Doc:  strings.Join(sf.doc, "\n"),
			}
			res = append(res, fs)
		}
		return res
	}

	encodeEnum := func(entry *Entry) []*spec.EnumValueSpec {
		var res []*spec.EnumValueSpec
		for _, ev := range entry.enumValues {
			evs := &spec.EnumValueSpec{
				Name:  ev.name,
				Value: ev.value,
				Doc:   strings.Join(ev.doc, "\n"),
			}
			res = append(res, evs)
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

				rs := &spec.RequestSpec{
					Method: entry.name,
					Caller: caller,
					Params: &spec.StructSpec{
						Fields: encodeStruct(params),
					},
					Result: &spec.StructSpec{
						Fields: encodeStruct(result),
					},
					Doc: strings.Join(params.doc, "\n"),
				}
				s.Requests = append(s.Requests, rs)
			case EntryKindNotification:
				ns := &spec.NotificationSpec{
					Method: entry.name,
					Params: &spec.StructSpec{
						Fields: encodeStruct(entry),
					},
					Doc: strings.Join(entry.doc, "\n"),
				}
				s.Notifications = append(s.Notifications, ns)
			case EntryKindType:
				switch entry.typeKind {
				case EntryTypeKindStruct:
					sts := &spec.StructTypeSpec{
						Name:   entry.name,
						Doc:    strings.Join(entry.doc, "\n"),
						Fields: encodeStruct(entry),
					}
					s.StructTypes = append(s.StructTypes, sts)
				case EntryTypeKindEnum:
					ets := &spec.EnumTypeSpec{
						Name:   entry.name,
						Doc:    strings.Join(entry.doc, "\n"),
						Values: encodeEnum(entry),
					}
					s.EnumTypes = append(s.EnumTypes, ets)
				}
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
