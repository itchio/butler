package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/structtag"
	"github.com/pkg/errors"
	"github.com/russross/blackfriday"
)

type scope struct {
	categories   map[string]*categoryInfo
	categoryList []string
	entries      map[string]*entryInfo
	gc           *generousContext
}

type categoryInfo struct {
	entries []*entryInfo
}

type callerInfo int

const (
	callerUnknown callerInfo = iota
	callerClient
	callerServer
)

type entryInfo struct {
	kind         entryKind
	typeKind     entryTypeKind
	gd           *ast.GenDecl
	typeSpec     *ast.TypeSpec
	tags         []string
	category     string
	doc          []string
	name         string
	typeName     string
	caller       callerInfo
	enumValues   []*enumValue
	structFields []*structField
}

type enumValue struct {
	name  string
	value string
	doc   []string
}

type structField struct {
	goName     string
	name       string
	typeString string
	typeNode   ast.Expr
	doc        []string
	optional   bool
}

type entryTypeKind int

const (
	entryTypeKindStruct entryTypeKind = iota
	entryTypeKindEnum
	entryTypeKindArrayAlias
	entryTypeKindAlias
	entryTypeKindInvalid
)

type entryKind int

const (
	entryKindParams entryKind = iota
	entryKindResult
	entryKindNotification
	entryKindType
	entryKindEnum
	entryKindArrayAlias
	entryKindAlias
	entryKindInvalid
)

func newScope(gc *generousContext) *scope {
	return &scope{
		categories:   make(map[string]*categoryInfo),
		categoryList: nil,
		entries:      make(map[string]*entryInfo),
		gc:           gc,
	}
}

func (s *scope) assimilate(pkg string, file string) error {
	log.Printf("Assimilating package (%s), file (%s)", pkg, file)

	absoluteFilePath := filepath.Join(getGoPackageDir(pkg), file)
	log.Printf("Parsing (%s)", absoluteFilePath)
	prefix := ""

	var fset token.FileSet
	f, err := parser.ParseFile(&fset, absoluteFilePath, nil, parser.ParseComments)
	if err != nil {
		return errors.Wrapf(err, "parsing %s", absoluteFilePath)
	}

	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			ts := asType(gd)
			if ts != nil {
				var typeKind = entryTypeKindInvalid
				if isStruct(ts) {
					typeKind = entryTypeKindStruct
				} else if isEnum(ts) {
					typeKind = entryTypeKindEnum
				} else if isArrayAlias(ts) {
					typeKind = entryTypeKindArrayAlias
				}

				if typeKind != entryTypeKindInvalid {
					tsName := ts.Name.Name

					kind := entryKindInvalid

					switch typeKind {
					case entryTypeKindEnum:
						kind = entryKindEnum
					case entryTypeKindArrayAlias:
						kind = entryKindArrayAlias
					case entryTypeKindStruct:
						switch true {
						case strings.HasSuffix(tsName, "Notification"):
							kind = entryKindNotification
						case strings.HasSuffix(tsName, "Params"):
							kind = entryKindParams
						case strings.HasSuffix(tsName, "Result"):
							kind = entryKindResult
						default:
							kind = entryKindType
						}
					}

					if kind != entryKindInvalid {
						category := "Miscellaneous"
						var tags []string
						var customName string
						var doc []string
						var caller = callerUnknown

						lines := getCommentLines(gd.Doc)
						if len(lines) > 0 {
							for _, line := range lines {
								tag, value := parseTag(line)
								switch tag {
								case "kind":
									switch value {
									case "type":
										kind = entryKindType
									default:
										log.Fatalf("Unknown @kind: (%s)", value)
									}
								case "name":
									customName = value
								case "category":
									category = value
								case "tags":
									tags = strings.Split(value, ", ")
								case "caller":
									switch value {
									case "server":
										caller = callerServer
									case "client":
										caller = callerClient
									default:
										panic(fmt.Sprintf("invalid caller specified for (%s): %s (must be server or client)", tsName, value))
									}
								default:
									doc = append(doc, line)
								}
							}

							// trim empty lines at the end
							for len(doc) > 0 {
								if doc[len(doc)-1] == "" {
									doc = doc[:len(doc)-1]
								} else {
									break
								}
							}
						}

						var name string
						if customName == "" {
							name = tsName
							switch kind {
							case entryKindParams:
								name = strings.TrimSuffix(name, "Params")
							case entryKindResult:
								name = strings.TrimSuffix(name, "Result")
							case entryKindNotification:
								name = strings.TrimSuffix(name, "Notification")
							}
						} else {
							name = customName
						}

						if kind == entryKindParams && caller == callerUnknown {
							panic(fmt.Sprintf("no caller specified for (%s) (must be server or client)", tsName))
						}

						e := &entryInfo{
							kind:     kind,
							typeKind: typeKind,
							name:     prefix + name,
							typeName: tsName,
							typeSpec: ts,
							gd:       gd,
							tags:     tags,
							category: category,
							doc:      doc,
							caller:   caller,
						}

						if typeKind == entryTypeKindStruct {
							st := ts.Type.(*ast.StructType)
							for _, sf := range st.Fields.List {
								if sf.Tag == nil {
									pos := fset.Position(sf.Pos())
									log.Fatalf("%s: %s.%s is untagged", pos, ts.Name.Name, sf.Names[0].Name)
								}

								tagValue := strings.TrimRight(strings.TrimLeft(sf.Tag.Value, "`"), "`")

								tags, err := structtag.Parse(tagValue)
								if err != nil {
									pos := fset.Position(sf.Pos())
									log.Fatalf("%s: For tag (%s): %s", pos, sf.Tag.Value, err.Error())
								}

								jsonTag, err := tags.Get("json")
								if err != nil {
									pos := fset.Position(sf.Pos())
									log.Fatalf("%s: %s.%s is lacking a 'json' tag", pos, ts.Name.Name, sf.Names[0].Name)
								}

								var optional = false
								var doc []string
								for _, line := range getCommentLines(sf.Doc) {
									if strings.Contains(line, "@optional") {
										optional = true
										continue
									}
									doc = append(doc, line)
								}

								e.structFields = append(e.structFields, &structField{
									goName:     sf.Names[0].Name,
									name:       jsonTag.Name,
									doc:        doc,
									typeString: typeToString(sf.Type),
									typeNode:   sf.Type,
									optional:   optional,
								})
							}
						}

						s.addEntry(category, e)

						s.entries[tsName] = e
					}
				}

			}
		}
	}

	// Now for enum values
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			if gd.Tok == token.CONST {
				hadValidType := false
				for _, spec := range gd.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						if typeid, ok := vs.Type.(*ast.Ident); ok {
							if enum, ok := s.entries[typeid.Name]; ok && enum.kind == entryKindEnum {
								hadValidType = true
								name := vs.Names[0]
								val := vs.Values[0]
								if bl, ok := val.(*ast.BasicLit); ok {
									if bl.Kind == token.STRING || bl.Kind == token.INT {
										shortName := strings.TrimPrefix(name.Name, enum.typeName)
										enum.enumValues = append(enum.enumValues, &enumValue{
											name:  shortName,
											value: bl.Value,
											doc:   getCommentLines(vs.Doc),
										})
									}
								}
							}
						} else {
							if hadValidType {
								// if we had a valid type once, one of the
								name := vs.Names[0].Name
								pos := fset.Position(vs.Names[0].Pos())
								log.Fatalf("%s: enum value %s is missing a type", pos, name)
							} else {
								break
							}
						}
					}
				}
			}
		}
	}

	for _, entry := range s.entries {
		if entry.kind == entryKindEnum && len(entry.enumValues) == 0 {
			entry.kind = entryKindAlias
			entry.typeKind = entryTypeKindAlias
		}
	}

	return nil
}

func (s *scope) linkType(typeName string, tip bool) string {
	return fmt.Sprintf("<code class=%#v>%s</code>", "typename", s.linkTypeInner(typeName, tip))
}

var doubleAtRe = regexp.MustCompile(`@@[\w]+`)

func (s *scope) markdownAll(input []string, tip bool) string {
	return s.markdown(strings.Join(input, "\n"), tip)
}

func (s *scope) markdown(input string, tip bool) string {
	buf := string(blackfriday.Run([]byte(input)))
	matches := doubleAtRe.FindAllString(buf, -1)
	for _, m := range matches {
		var typeName = strings.TrimPrefix(m, "@@")
		buf = strings.Replace(buf, m, s.linkType(typeName, tip), -1)
	}
	return buf
}

var mapRegexp = regexp.MustCompile(`Map<([^,]+),\s*([^>]+)>`)

func (s *scope) linkTypeInner(typeName string, tip bool) string {
	mapMatches := mapRegexp.FindStringSubmatch(typeName)
	if len(mapMatches) > 0 {
		return fmt.Sprintf(`Map&lt;%s, %s&gt;`, s.linkTypeInner(mapMatches[1], tip), s.linkTypeInner(mapMatches[2], tip))
	}

	if strings.HasSuffix(typeName, "[]") {
		return s.linkTypeInner(strings.TrimSuffix(typeName, "[]"), tip) + "[]"
	}

	if entry, ok := s.entries[typeName]; ok {
		var className = "type"
		if tip {
			sel := fmt.Sprintf("#%s__TypeHint", entry.typeName)
			return fmt.Sprintf(`<span class=%#v data-tip-selector="%s">%s</span>`, className, sel, entry.name)
		}
		return fmt.Sprintf(`<span class=%#v>%s</span>`, className, entry.name)
	}

	return fmt.Sprintf(`<span class=%#v>%s</span>`, "type builtin-type", typeName)
}

func (s *scope) addEntry(category string, e *entryInfo) {
	e.category = category
	cat, ok := s.categories[category]
	if !ok {
		cat = &categoryInfo{}
		s.categoryList = append(s.categoryList, category)
		s.categories[category] = cat
	}

	cat.entries = append(cat.entries, e)
}

func (s *scope) findEntry(name string) *entryInfo {
	return s.entries[name]
}
