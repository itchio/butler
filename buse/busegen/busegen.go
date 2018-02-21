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

	dumpStruct := func(obj *ast.Object) {
		ts := obj.Decl.(*ast.TypeSpec)
		st := ts.Type.(*ast.StructType)
		fl := st.Fields.List

		if len(fl) == 0 {
			line("*empty*")
			return
		}
		line("Name | Type | JSON Tag")
		line("--- | --- | ---")
		for _, sf := range fl {
			name := sf.Names[0].Name
			tag := sf.Tag.Value
			line("%s | %s | %s", name, sf.Type, tag)
		}
	}

	var fset token.FileSet
	f, err := parser.ParseFile(&fset, "../types.go", nil, parser.ParseComments)
	if err != nil {
		return errors.Wrap(err, 0)
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
		return i < j
	})

	sort.Strings(requestNames)
	sort.Strings(notificationNames)

	for _, name := range requestNames {
		line("## %s", name)
		line("")

		params := f.Scope.Objects[name+"Params"]
		result := f.Scope.Objects[name+"Result"]

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
