package main

import (
	"encoding/json"
	"strings"

	"github.com/itchio/butler/butlerd/generous/spec"
	"github.com/pkg/errors"
)

func (gc *generousContext) generateSpec() error {
	gc.task("Generating JSON spec")

	doc := gc.newGenerousRelativeDoc("spec/butlerd.json")

	s := &spec.Spec{}

	scope := newScope(gc)
	scope.assimilateAll()

	encodeStruct := func(entry *entryInfo) []*spec.FieldSpec {
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

	encodeEnum := func(entry *entryInfo) []*spec.EnumValueSpec {
		var res []*spec.EnumValueSpec
		for _, ev := range entry.enumValues {
			evs := &spec.EnumValueSpec{
				Name:  ev.name,
				Value: ev.jsonValue,
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
			case entryKindParams:
				params := entry
				result := scope.findEntry(strings.TrimSuffix(entry.typeName, "Params") + "Result")
				var caller string
				switch entry.caller {
				case callerClient:
					caller = "client"
				case callerServer:
					caller = "server"
				}

				rs := &spec.RequestSpec{
					Method:     entry.name,
					Caller:     caller,
					Deprecated: entry.deprecated,
					Params: &spec.StructSpec{
						Fields: encodeStruct(params),
					},
					Result: &spec.StructSpec{
						Fields: encodeStruct(result),
					},
					Doc: strings.Join(params.doc, "\n"),
				}
				s.Requests = append(s.Requests, rs)
			case entryKindResult:
				sts := &spec.StructTypeSpec{
					Name:       entry.typeName,
					Doc:        strings.Join(entry.doc, "\n"),
					Deprecated: entry.deprecated,
					Fields:     encodeStruct(entry),
				}
				s.StructTypes = append(s.StructTypes, sts)
			case entryKindNotification:
				ns := &spec.NotificationSpec{
					Method:     entry.name,
					Deprecated: entry.deprecated,
					Params: &spec.StructSpec{
						Fields: encodeStruct(entry),
					},
					Doc: strings.Join(entry.doc, "\n"),
				}
				s.Notifications = append(s.Notifications, ns)
			case entryKindType:
				switch entry.typeKind {
				case entryTypeKindStruct:
					sts := &spec.StructTypeSpec{
						Name:       entry.name,
						Doc:        strings.Join(entry.doc, "\n"),
						Deprecated: entry.deprecated,
						Fields:     encodeStruct(entry),
					}
					s.StructTypes = append(s.StructTypes, sts)
				}
			case entryKindEnum:
				ets := &spec.EnumTypeSpec{
					Name:   entry.name,
					Doc:    strings.Join(entry.doc, "\n"),
					Values: encodeEnum(entry),
				}
				s.EnumTypes = append(s.EnumTypes, ets)
			}
		}
	}

	js, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	doc.line("%s", string(js))
	doc.commit("")
	doc.write()

	return nil
}
