package main

import (
	"fmt"
	"go/ast"
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

	linkType := func(typeName string) string {
		return fmt.Sprintf("[`%s`](#%s-type)", typeName, linkify(typeName))
	}

	dumpStruct := func(header string, gd *ast.GenDecl) {
		if gd == nil {
			return
		}

		ts := gd.Specs[0].(*ast.TypeSpec)
		st := ts.Type.(*ast.StructType)
		fl := st.Fields.List

		if len(fl) == 0 {
			return
		}

		line("")
		line("%s: ", header)
		line("")

		// line("Name | Type | Description")
		// line("--- | --- | ---")
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
			// line("**%s** | %s | %s", jsonTag.Name, linkType(typeToString(sf.Type)), comment)
			// line("  * `%s` %s — %s", jsonTag.Name, linkType(typeToString(sf.Type)), comment)
			line("  * `%s` %s  ", jsonTag.Name, linkType(typeToString(sf.Type)))
			line("    %s", comment)
		}
		line("")
	}

	scope := newScope()
	err = scope.Assimilate("", "../types.go")
	if err != nil {
		return errors.Wrap(err, 0)
	}

	renderHeader := func(entry *Entry) {
		var kindString string
		switch entry.kind {
		case EntryKindParams:
			kindString = `<em class="request">Request</em>`
		case EntryKindNotification:
			kindString = `<em class="notification">Notification</em>`
		case EntryKindType:
			kindString = `<em class="type">Type</em>`
		}

		line("### %s %s", entry.name, kindString)
		if len(entry.tags) > 0 {
			line("<p class='tags'>")
			for _, tag := range entry.tags {
				line("<em>%s</em>", tag)
			}
			line("</p>")
		}

		line("")
		line(entry.doc)
		line("")
	}

	renderRequest := func(params *Entry) {
		paramsName := asType(params.gd).Name.Name
		resultName := strings.TrimSuffix(paramsName, "Params") + "Result"

		result := scope.FindStruct(resultName)

		renderHeader(params)
		dumpStruct("Parameters", params.gd)
		dumpStruct("Result", result)
	}

	renderNotification := func(entry *Entry) {
		renderHeader(entry)
		dumpStruct("Payload", entry.gd)
	}

	renderType := func(entry *Entry) {
		renderHeader(entry)
		dumpStruct("Fields", entry.gd)
	}

	line("")
	line("# Messages")
	line("")

	for _, category := range scope.categoryList {
		line("")
		line("## %s", category)
		line("")

		cat := scope.categories[category]
		for _, entry := range cat.entries {
			switch entry.kind {
			case EntryKindParams:
				renderRequest(entry)
			case EntryKindNotification:
				renderNotification(entry)
			case EntryKindType:
				renderType(entry)
			}
		}
	}

	commit("{{EVERYTHING}}")

	err = ioutil.WriteFile(outPath, []byte(doc), 0644)
	if err != nil {
		return errors.Wrap(err, 0)
	}

	return nil
}
