package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"strings"

	"github.com/go-errors/errors"
)

type Scope struct {
	categories   map[string]*Category
	categoryList []string
	structs      map[string]*ast.GenDecl
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
	}
}

func (s *Scope) Assimilate(prefix string, path string) error {
	var fset token.FileSet
	f, err := parser.ParseFile(&fset, path, nil, parser.ParseComments)
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
				}
			}
		}
	}
	return nil
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
