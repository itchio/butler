package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/structtag"
	"github.com/go-errors/errors"
)

func main() {
	log.SetFlags(0)

	err := doMain()
	if err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	wd, err := os.Getwd()
	if err != nil {
		return errors.Wrap(err, 0)
	}
	log.Printf("Working directory: (%s)", wd)

	layoutPath := filepath.Join(wd, "layout.md")
	log.Printf("Reading layout from: (%s)", layoutPath)
	layoutBytes, err := ioutil.ReadFile(layoutPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	doc := string(layoutBytes)
	buffer := ""

	outPath := filepath.Join(wd, "docs", "README.md")
	log.Printf("Out path: (%s)", outPath)

	line := func(msg string, args ...interface{}) {
		buffer += fmt.Sprintf(msg, args...)
		buffer += "\n"
	}

	commit := func(name string) {
		doc = strings.Replace(doc, name, buffer, 1)
		buffer = ""
	}

	jsonType := func(goType string) string {
		switch goType {
		case "string":
			return "string"
		case "int64", "float64":
			return "number"
		case "bool":
			return "boolean"
		default:
			return goType
		}
	}

	var typeToString func(e ast.Expr) string

	typeToString = func(e ast.Expr) string {
		switch node := e.(type) {
		case *ast.Ident:
			return jsonType(node.Name)
		case *ast.StarExpr:
			return typeToString(node.X)
		case *ast.SelectorExpr:
			return typeToString(node.X) + "." + node.Sel.Name
		case *ast.ArrayType:
			return typeToString(node.Elt) + "[]"
		case *ast.MapType:
			return "Map<" + typeToString(node.Key) + ", " + typeToString(node.Value) + ">"
		default:
			return fmt.Sprintf("%#v", node)
		}
	}

	getComment := func(doc *ast.CommentGroup, separator string) string {
		if doc == nil {
			return "*undocumented*"
		}

		comment := ""
		for _, line := range doc.List {
			comment += strings.TrimPrefix(line.Text, "// ")
			comment += separator
		}
		return comment
	}

	dumpStruct := func(gd *ast.GenDecl) {
		ts := gd.Specs[0].(*ast.TypeSpec)
		st := ts.Type.(*ast.StructType)
		fl := st.Fields.List

		if len(fl) == 0 {
			line("*empty*")
			return
		}

		line("Name | Type | Description")
		line("--- | --- | ---")
		for _, sf := range fl {
			tagValue := strings.TrimRight(strings.TrimLeft(sf.Tag.Value, "`"), "`")

			tags, err := structtag.Parse(tagValue)
			if err != nil {
				log.Fatalf("For tag (%s): %s", sf.Tag.Value, err.Error())
			}

			jsonTag, err := tags.Get("json")
			if err != nil {
				panic(err)
			}

			comment := getComment(sf.Doc, " ")
			line("**%s** | `%s` | %s", jsonTag.Name, typeToString(sf.Type), comment)
		}
	}

	var fset token.FileSet
	f, err := parser.ParseFile(&fset, "../types.go", nil, parser.ParseComments)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	var paramDecls []*ast.GenDecl
	var notificationDecls []*ast.GenDecl
	var typeDecls []*ast.GenDecl

	asType := func(gd *ast.GenDecl) *ast.TypeSpec {
		for _, spec := range gd.Specs {
			if ts, ok := spec.(*ast.TypeSpec); ok {
				return ts
			}
		}
		return nil
	}

	isStruct := func(ts *ast.TypeSpec) bool {
		if ts == nil {
			return false
		}

		_, ok := ts.Type.(*ast.StructType)
		return ok
	}

	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			ts := asType(gd)
			if ts != nil && isStruct(ts) {
				name := ts.Name.Name
				switch true {
				case strings.HasSuffix(name, "Params"):
					paramDecls = append(paramDecls, gd)
				case strings.HasSuffix(name, "Notification"):
					notificationDecls = append(notificationDecls, gd)
				case strings.HasSuffix(name, "Result"):
					// ignore
				default:
					typeDecls = append(typeDecls, gd)
				}
			}
		}
	}

	findStruct := func(name string) *ast.GenDecl {
		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok {
				ts := asType(gd)
				if ts != nil && isStruct(ts) {
					if ts.Name.Name == name {
						return gd
					}
				}
			}
		}
		return nil
	}

	for _, params := range paramDecls {
		name := asType(params).Name.Name
		name = strings.TrimSuffix(name, "Params")

		line("## %s", name)
		line("")

		result := findStruct(name + "Result")

		comment := getComment(params.Doc, "\n")
		line(comment)

		line("")
		line("Parameters:")
		line("")

		dumpStruct(params)

		line("")
		line("Result:")
		line("")

		if result == nil {
			line("*empty*")
		} else {
			dumpStruct(result)
		}
		line("")
	}

	commit("{{REQUESTS}}")

	for _, notification := range notificationDecls {
		name := asType(notification).Name.Name
		name = strings.TrimSuffix(name, "Notification")

		line("## %s", name)
		line("")

		comment := getComment(notification.Doc, "\n")
		line(comment)

		line("")
		line("Payload:")
		line("")

		dumpStruct(notification)

		line("")
	}

	commit("{{NOTIFICATIONS}}")

	for _, typ := range typeDecls {
		name := asType(typ).Name.Name

		line("## %s", name)
		line("")

		comment := getComment(typ.Doc, "\n")
		line(comment)

		line("")
		line("Fields:")
		line("")

		dumpStruct(typ)

		line("")
	}

	commit("{{TYPES}}")

	err = ioutil.WriteFile(outPath, []byte(doc), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
