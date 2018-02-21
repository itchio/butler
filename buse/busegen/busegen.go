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
	"sort"
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
			return "(null doc)"
		}

		comment := ""
		for _, line := range doc.List {
			comment += strings.TrimPrefix(line.Text, "// ")
			comment += separator
		}
		return comment
	}

	dumpStruct := func(obj *ast.Object) {
		ts := obj.Decl.(*ast.TypeSpec)
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

	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range gd.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					log.Printf("%s: ", ts.Name.Name)
					log.Printf("Docs: %s", getComment(gd.Doc, " "))
				}
			}
		}
	}

	var requestNames []string
	var notificationNames []string

	for declName, object := range f.Scope.Objects {
		if object.Kind == ast.Typ {
			switch true {
			case strings.HasSuffix(declName, "Params"):
				name := strings.TrimSuffix(declName, "Params")
				requestNames = append(requestNames, name)
			case strings.HasSuffix(declName, "Notification"):
				name := strings.TrimSuffix(declName, "Notification")
				notificationNames = append(notificationNames, name)
			}
		}
	}
	sort.Slice(requestNames, func(i, j int) bool {
		a := f.Scope.Objects[requestNames[i]+"Params"]
		b := f.Scope.Objects[requestNames[j]+"Params"]
		return a.Pos() < b.Pos()
	})
	sort.Slice(notificationNames, func(i, j int) bool {
		a := f.Scope.Objects[notificationNames[i]+"Notification"]
		b := f.Scope.Objects[notificationNames[j]+"Notification"]
		return a.Pos() < b.Pos()
	})

	for _, name := range requestNames {
		line("## %s", name)
		line("")

		params := f.Scope.Objects[name+"Params"]
		result := f.Scope.Objects[name+"Result"]

		line("%#v", params)
		ts := params.Decl.(*ast.TypeSpec)

		line("")
		line("Comment:")
		line("")
		comment := getComment(ts.Comment, "\n")
		line(comment)

		line("")
		line("Doc:")
		line("")
		comment = getComment(ts.Doc, "\n")
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

	for _, name := range notificationNames {
		line("## %s", name)
		line("")

		notif := f.Scope.Objects[name+"Notification"]

		line("")
		line("Payload:")
		line("")

		dumpStruct(notif)

		line("")
	}

	commit("{{NOTIFICATIONS}}")

	err = ioutil.WriteFile(outPath, []byte(doc), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
