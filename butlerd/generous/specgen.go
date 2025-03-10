package main

import (
	"encoding/json"
	"go/ast"
	"strings"

	"github.com/itchio/butler/butlerd/generous/spec"
	"github.com/pkg/errors"
)

func resolveTypeName(typeName string, aliasMap map[string]*ast.Ident, aliasIsArrayMap map[string]bool) string {
	isArray := strings.HasSuffix(typeName, "[]")
	nonArray := strings.TrimSuffix(typeName, "[]")
	if target, ok :=  aliasMap[nonArray]; ok {
		resolved := target.Name
		// Does not consider an array of array aliases
		if aliasIsArrayMap[nonArray] || isArray {
			resolved = resolved + "[]"
		}
		return resolved
	}
	return typeName
}

func resolveEntryAliases(entry *entryInfo, aliasMap map[string]*ast.Ident, aliasIsArrayMap map[string]bool) {
	for _, field := range entry.structFields {
		field.name = resolveTypeName(field.name, aliasMap, aliasIsArrayMap)
	}
}

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
				Value: ev.value,
				Doc:   strings.Join(ev.doc, "\n"),
			}
			res = append(res, evs)
		}
		return res
	}

	aliasMap := make(map[string]*ast.Ident)
	aliasIsArrayMap := make(map[string]bool)

	// Replace aliases with the real type
	for _, category := range scope.categoryList {
		for _, entry := range scope.categories[category].entries {
			isArray := false
			switch entry.kind {
			case entryKindArrayAlias:
				isArray = true
				fallthrough
			case entryKindAlias:
				if target, ok := entry.typeSpec.Type.(*ast.Ident); ok {
					aliasMap[entry.typeName] = target
					aliasIsArrayMap[entry.typeName] = isArray
				}
			}
		}
	}

	for _, category := range scope.categoryList {
		for _, entry := range scope.categories[category].entries {
			isArray := false
			switch entry.kind {
			case entryKindArrayAlias:
				isArray = true
				fallthrough
			case entryKindAlias:
				if target, ok := entry.typeSpec.Type.(*ast.Ident); ok {
					aliasMap[entry.typeName] = target
					aliasIsArrayMap[entry.typeName] = isArray
				}
			}
		}
	}

	for _, category := range scope.categoryList {
		cat := scope.categories[category]
		for _, entry := range cat.entries {
			resolveEntryAliases(entry, aliasMap, aliasIsArrayMap)
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
				sts := &spec.StructTypeSpec {
					Name: entry.typeName,
					Doc:    strings.Join(entry.doc, "\n"),
					Fields: encodeStruct(entry),
				}
				s.StructTypes = append(s.StructTypes, sts)
			case entryKindResult:
				sts := &spec.StructTypeSpec {
					Name: entry.typeName,
					Doc:    strings.Join(entry.doc, "\n"),
					Fields: encodeStruct(entry),
				}
				s.StructTypes = append(s.StructTypes, sts)
			case entryKindNotification:
				ns := &spec.NotificationSpec{
					Method: entry.name,
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
							Name:   entry.name,
							Doc:    strings.Join(entry.doc, "\n"),
							Fields: encodeStruct(entry),
						}
						s.StructTypes = append(s.StructTypes, sts)
					case entryTypeKindEnum:
						ets := &spec.EnumTypeSpec{
							Name:   entry.name,
							Doc:    strings.Join(entry.doc, "\n"),
							Values: encodeEnum(entry),
						}
						s.EnumTypes = append(s.EnumTypes, ets)
					case entryTypeKindArrayAlias:
						// Handled by encodeStruct
					case entryTypeKindAlias:
						// Handled by encodeStruct
					case entryTypeKindInvalid:

				}
			case entryKindEnum:
				ets := &spec.EnumTypeSpec{
					Name:   entry.name,
					Doc:    strings.Join(entry.doc, "\n"),
					Values: encodeEnum(entry),
				}
				s.EnumTypes = append(s.EnumTypes, ets)
			case entryKindArrayAlias:
				// Handled by encodeStruct
			case entryKindAlias:
				// Handled by encodeStruct
			case entryKindInvalid:
			}
		}
	}

	js, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	doc.line(string(js))
	doc.commit("")
	doc.write()

	return nil
}
