package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-errors/errors"
)

type Scope struct {
	categories   map[string]*Category
	categoryList []string
	structs      map[string]*ast.GenDecl
	entries      map[string]*Entry
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
	kind     EntryKind
	gd       *ast.GenDecl
	tags     []string
	category string
	doc      string
	name     string
	caller   Caller
	pkg      string
	pkgName  string
}

type EntryKind int

const (
	EntryKindParams EntryKind = iota
	EntryKindResult
	EntryKindNotification
	EntryKindType
	EntryKindInvalid
)

func newScope() *Scope {
	return &Scope{
		categories:   make(map[string]*Category),
		categoryList: nil,
		structs:      make(map[string]*ast.GenDecl),
		entries:      make(map[string]*Entry),
	}
}

const butlerPkg = "github.com/itchio/butler"

func (s *Scope) Assimilate(pkg string, file string) error {
	log.Printf("Assimilating package (%s), file (%s)", pkg, file)

	wd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, 0)
	}
	var rootPath = filepath.Join(wd, "..", "..")

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
			if ts != nil && isStruct(ts) {
				tsName := ts.Name.Name
				s.structs[tsName] = gd
				kind := EntryKindInvalid

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

				if kind != EntryKindInvalid {
					category := "Miscellaneous"
					var tags []string
					var customName string
					var doc string
					var caller = CallerUnknown

					lines := getCommentLines(gd.Doc)
					if len(lines) > 0 {
						var outlines []string
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
								outlines = append(outlines, line)
							}
						}

						doc = strings.Join(outlines, "\n")
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
						name:     prefix + name,
						gd:       gd,
						tags:     tags,
						category: category,
						doc:      doc,
						caller:   caller,
					}
					s.AddEntry(category, e)

					s.entries[tsName] = e
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

func (s *Scope) LinkTypeInner(typeName string, tip bool) string {
	if strings.Contains(typeName, "<") {
		return typeName
	}

	if strings.HasSuffix(typeName, "[]") {
		return s.LinkTypeInner(strings.TrimSuffix(typeName, "[]"), tip) + "[]"
	}

	if gd, ok := s.structs[typeName]; ok {
		if tip {
			ts := gd.Specs[0].(*ast.TypeSpec)
			sel := fmt.Sprintf("#%s__TypeHint", ts.Name.Name)
			return fmt.Sprintf(`<span class="struct-type" data-tip-selector="%s">%s</span>`, sel, typeName)
		} else {
			return fmt.Sprintf(`<span class="struct-type">%s</span>`, typeName)
		}
	}

	return typeName
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

func (s *Scope) FindStruct(name string) *ast.GenDecl {
	return s.structs[name]
}

func (s *Scope) FindEntry(name string) *Entry {
	return s.entries[name]
}
