package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/structtag"
	"github.com/go-errors/errors"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

type Scope struct {
	categories   map[string]*Category
	categoryList []string
	structs      map[string]*Entry
	enums        map[string]*Entry
	entries      map[string]*Entry
	bc           *BuseContext
}

type Category struct {
	entries []*Entry
}

type Caller int

const (
	CallerUnknown Caller = iota
	CallerClient
	CallerServer
)

type Entry struct {
	kind         EntryKind
	typeKind     EntryTypeKind
	gd           *ast.GenDecl
	typeSpec     *ast.TypeSpec
	tags         []string
	category     string
	doc          []string
	name         string
	typeName     string
	caller       Caller
	pkg          string
	pkgName      string
	enumValues   []*EnumValue
	structFields []*StructField
}

type EnumValue struct {
	name  string
	value string
	doc   []string
}

type StructField struct {
	goName     string
	name       string
	typeString string
	typeNode   ast.Expr
	doc        []string
	optional   bool
}

type EntryTypeKind int

const (
	EntryTypeKindStruct EntryTypeKind = iota
	EntryTypeKindEnum
	EntryTypeKindInvalid
)

type EntryKind int

const (
	EntryKindParams EntryKind = iota
	EntryKindResult
	EntryKindNotification
	EntryKindType
	EntryKindEnum
	EntryKindInvalid
)

func newScope(bc *BuseContext) *Scope {
	return &Scope{
		categories:   make(map[string]*Category),
		categoryList: nil,
		entries:      make(map[string]*Entry),
		bc:           bc,
	}
}

const butlerPkg = "github.com/itchio/butler"

func (s *Scope) Assimilate(pkg string, file string) error {
	log.Printf("Assimilating package (%s), file (%s)", pkg, file)

	var rootPath = filepath.Join(s.bc.Dir, "..", "..")

	var relativePath string
	if strings.HasPrefix(pkg, butlerPkg) {
		relativePath = strings.TrimPrefix(pkg, butlerPkg)
	} else {
		relativePath = path.Join("vendor", pkg)
	}

	absoluteFilePath := filepath.Join(rootPath, filepath.FromSlash(relativePath), file)
	log.Printf("Parsing (%s)", absoluteFilePath)
	prefix := ""

	var fset token.FileSet
	f, err := parser.ParseFile(&fset, absoluteFilePath, nil, parser.ParseComments)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			ts := asType(gd)
			if ts != nil {
				var typeKind = EntryTypeKindInvalid
				if isStruct(ts) {
					typeKind = EntryTypeKindStruct
				} else if isEnum(ts) {
					typeKind = EntryTypeKindEnum
				}

				if typeKind != EntryTypeKindInvalid {
					tsName := ts.Name.Name

					kind := EntryKindInvalid

					switch typeKind {
					case EntryTypeKindEnum:
						kind = EntryKindEnum
					case EntryTypeKindStruct:
						switch true {
						case strings.HasSuffix(tsName, "Notification"):
							kind = EntryKindNotification
						case strings.HasSuffix(tsName, "Params"):
							kind = EntryKindParams
						case strings.HasSuffix(tsName, "Result"):
							kind = EntryKindResult
						default:
							kind = EntryKindType
						}
					}

					if kind != EntryKindInvalid {
						category := "Miscellaneous"
						var tags []string
						var customName string
						var doc []string
						var caller = CallerUnknown

						lines := getCommentLines(gd.Doc)
						if len(lines) > 0 {
							for _, line := range lines {
								tag, value := parseTag(line)
								switch tag {
								case "kind":
									switch value {
									case "type":
										kind = EntryKindType
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
										caller = CallerServer
									case "client":
										caller = CallerClient
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
							case EntryKindParams:
								name = strings.TrimSuffix(name, "Params")
							case EntryKindResult:
								name = strings.TrimSuffix(name, "Result")
							case EntryKindNotification:
								name = strings.TrimSuffix(name, "Notification")
							}
						} else {
							name = customName
						}

						if kind == EntryKindParams && caller == CallerUnknown {
							panic(fmt.Sprintf("no caller specified for (%s) (must be server or client)", tsName))
						}

						e := &Entry{
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

						if typeKind == EntryTypeKindStruct {
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
									log.Fatalf("For tag (%s): %s", pos, sf.Tag.Value, err.Error())
								}

								jsonTag, err := tags.Get("json")
								if err != nil {
									panic(err)
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

								e.structFields = append(e.structFields, &StructField{
									goName:     sf.Names[0].Name,
									name:       jsonTag.Name,
									doc:        doc,
									typeString: typeToString(sf.Type),
									typeNode:   sf.Type,
									optional:   optional,
								})
							}
						}

						s.AddEntry(category, e)

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
							if enum, ok := s.entries[typeid.Name]; ok && enum.kind == EntryKindEnum {
								hadValidType = true
								name := vs.Names[0]
								val := vs.Values[0]
								if bl, ok := val.(*ast.BasicLit); ok {
									if bl.Kind == token.STRING || bl.Kind == token.INT {
										shortName := strings.TrimPrefix(name.Name, enum.typeName)
										enum.enumValues = append(enum.enumValues, &EnumValue{
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

	return nil
}

func (s *Scope) LinkType(typeName string, tip bool) string {
	return fmt.Sprintf("<code class=%#v>%s</code>", "typename", s.LinkTypeInner(typeName, tip))
}

var builtinTypes = map[string]struct{}{
	"number":  struct{}{},
	"string":  struct{}{},
	"boolean": struct{}{},
}

var doubleAtRe = regexp.MustCompile(`@@[\w]+`)

func (s *Scope) MarkdownAll(input []string, tip bool) string {
	return s.Markdown(strings.Join(input, "\n"), tip)
}

func (s *Scope) Markdown(input string, tip bool) string {
	buf := string(blackfriday.Run([]byte(input)))
	matches := doubleAtRe.FindAllString(buf, -1)
	for _, m := range matches {
		var typeName = strings.TrimPrefix(m, "@@")
		buf = strings.Replace(buf, m, s.LinkType(typeName, tip), -1)
	}
	return buf
}

var mapRegexp = regexp.MustCompile(`Map<([^,]+),\s*([^>]+)>`)

func (s *Scope) LinkTypeInner(typeName string, tip bool) string {
	mapMatches := mapRegexp.FindStringSubmatch(typeName)
	if len(mapMatches) > 0 {
		return fmt.Sprintf(`Map&lt;%s, %s&gt;`, s.LinkTypeInner(mapMatches[1], tip), s.LinkTypeInner(mapMatches[2], tip))
	}

	if strings.HasSuffix(typeName, "[]") {
		return s.LinkTypeInner(strings.TrimSuffix(typeName, "[]"), tip) + "[]"
	}

	if entry, ok := s.entries[typeName]; ok {
		var className string
		switch entry.typeKind {
		case EntryTypeKindStruct:
			switch entry.kind {
			case EntryKindParams:
				switch entry.caller {
				case CallerClient:
					className = "type request-client-caller"
				case CallerServer:
					className = "type request-server-caller"
				}
			case EntryKindNotification:
				className = "type notification"
			default:
				className = "type struct-type"
			}
		case EntryTypeKindEnum:
			className = "type enum-type"
		}

		if tip {
			sel := fmt.Sprintf("#%s__TypeHint", entry.typeName)
			return fmt.Sprintf(`<span class=%#v data-tip-selector="%s">%s</span>`, className, sel, entry.name)
		}
		return fmt.Sprintf(`<span class=%#v>%s</span>`, className, entry.name)
	}

	return fmt.Sprintf(`<span class=%#v>%s</span>`, "type builtin-type", typeName)
}

func (s *Scope) AddEntry(category string, e *Entry) {
	e.category = category
	cat, ok := s.categories[category]
	if !ok {
		cat = &Category{}
		s.categoryList = append(s.categoryList, category)
		s.categories[category] = cat
	}

	cat.entries = append(cat.entries, e)
}

func (s *Scope) FindEntry(name string) *Entry {
	return s.entries[name]
}
