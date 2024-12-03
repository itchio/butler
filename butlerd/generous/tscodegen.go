package main

import (
	"log"
	"strings"
)

func (gc *generousContext) generateTsCode(outPath string) error {
	gc.task("Generating typescript bindings")

	doc := gc.newPathDoc(outPath)

	doc.line("")
	doc.line("// These bindings were generated by generous")
	doc.line("// See <https://docs.itch.zone/butlerd/master/> for a human-friendly documentation")

	doc.line("")
	doc.line("import { createRequest, createNotification } from %#v;", gc.SupportPath)

	doc.line("")
	doc.line("// Type alias for RFC3339-nano date strings")
	doc.line("export type RFCDate = string;")

	scope := newScope(gc)
	scope.assimilateAll()

	bindType := func(entry *entryInfo) {
		doc.line("")
		doc.line("/**")
		switch entry.kind {
		case entryKindParams:
			doc.line(" * Params for %s", entry.name)
		case entryKindResult:
			params := scope.findEntry(strings.TrimSuffix(entry.typeName, "Result") + "Params")
			if params != nil {
				doc.line(" * Result for %s", params.name)
			}
		case entryKindNotification:
			doc.line(" * Payload for %s", entry.name)
		default:
			if len(entry.doc) == 0 {
				doc.line(" * undocumented")
			} else {
				for _, line := range entry.doc {
					doc.line(" * %s", line)
				}
			}
		}
		doc.line(" */")
		switch entry.typeKind {
		case entryTypeKindStruct:
			doc.line("export interface %s {", entry.typeName)
			if len(entry.structFields) == 0 {
				doc.line("  // no fields")
			} else {
				for _, sf := range entry.structFields {
					if len(sf.doc) == 0 {
						doc.line("  /** undocumented */")
					} else if len(sf.doc) == 1 {
						doc.line("  /** %s */", sf.doc[0])
					} else {
						doc.line("  /**")
						for _, line := range sf.doc {
							doc.line("   * %s", line)
						}
						doc.line("   */")
					}

					var optionalMarker = ""
					if sf.optional {
						optionalMarker = "?"
					}

					doc.line("  %s%s: %s;", sf.name, optionalMarker, sf.typeString)
				}
			}

			doc.line("}")
		case entryTypeKindArrayAlias:
			doc.line("export type %s = %s;", entry.typeName, typeToString(entry.typeSpec.Type))
		case entryTypeKindEnum:
			doc.line("export enum %s {", entry.typeName)
			for _, val := range entry.enumValues {
				for _, line := range val.doc {
					doc.line("  // %s", line)
				}
				// special case for "386", woo
				name := val.name
				if strings.ContainsAny(name[0:1], "0123456789") {
					name = "_" + name
				}
				doc.line("  %s = %s,", name, val.value)
			}
			doc.line("}")
		case entryTypeKindAlias:
			doc.line("export type %s = %s;", entry.typeName, typeToString(entry.typeSpec.Type))
		}
	}

	for _, category := range scope.categoryList {
		cat := scope.categories[category]
		for _, entry := range cat.entries {
			bindType(entry)

			switch entry.kind {
			case entryKindResult:
				paramsTypeName := strings.TrimSuffix(entry.typeName, "Result") + "Params"
				resultTypeName := entry.typeName
				params := scope.findEntry(paramsTypeName)
				if params == nil {
					log.Printf("Warning: %q doesn't have a Params equivalent", entry.typeName)
					continue
				}
				method := params.name
				symbolName := strings.Replace(method, ".", "", -1)

				doc.line("")
				doc.line("/**")
				if len(params.doc) == 0 {
					doc.line(" * undocumented")
				} else {
					for _, line := range params.doc {
						doc.line(" * %s", line)
					}
				}
				doc.line(" */")
				doc.line("export const %s = createRequest<", symbolName)
				doc.line("  %s,", paramsTypeName)
				doc.line("  %s", resultTypeName)
				doc.line(">(%#v);", method)
			case entryKindNotification:
				method := entry.name
				symbolName := strings.Replace(method, ".", "", -1)

				doc.line("")
				doc.line("")
				doc.line("/**")
				if len(entry.doc) == 0 {
					doc.line(" * undocumented")
				} else {
					for _, line := range entry.doc {
						doc.line(" * %s", line)
					}
				}
				doc.line(" */")
				doc.line("export const %s = ", symbolName)
				doc.line("	createNotification<%s>(%#v);", entry.typeName, method)
			}
		}
	}

	doc.commit("")
	doc.write()

	return nil
}
