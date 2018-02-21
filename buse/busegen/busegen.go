package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

	outPath := filepath.Join(wd, "docs", "README.md")
	log.Printf("Out path: (%s)", outPath)

	out, err := os.Create(outPath)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	defer out.Close()

	line := func(msg string, args ...interface{}) {
		out.WriteString(fmt.Sprintf(msg, args...))
		out.WriteString("\n")
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
	sort.Strings(requestNames)
	sort.Strings(notificationNames)

	line("# buse")
	line("")
	line("> butler's JSON-RPC 2.0 service documentation")

	line("")
	line("# Requests")
	line("")
	line("Requests are essentially procedure calls: they're made asynchronously, and")
	line("a result is sent asynchronously. They may also fail, in which case")
	line("you get an error back, with details.")
	line("")
	line("Some requests may complete almost instantly, and have an empty result")
	line("Still, waiting for the result lets you know that the peer has received")
	line("the request and processed it successfully.")
	line("")
	line("Some requests are made by the client to butler (like CheckUpdate),")
	line("others are made from butler to the client (like AllowSandboxSetup)")

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

	line("")
	line("# Notifications")
	line("")

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

	return nil
}
