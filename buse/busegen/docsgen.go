package main

import (
	"fmt"
	"go/ast"
	"log"
	"sort"
	"strings"

	"github.com/fatih/structtag"
	blackfriday "gopkg.in/russross/blackfriday.v2"
)

func (bc *BuseContext) GenerateDocs() error {
	bc.Task("Generating docs")

	doc := bc.NewDoc("docs/README.md")
	doc.Load(bc.ReadFile("layout.md"))

	scope := newScope()
	must(scope.Assimilate("github.com/itchio/butler/buse", "types.go"))
	must(scope.Assimilate("github.com/itchio/go-itchio", "types.go"))

	dumpStruct := func(header string, entry *Entry, showDesc bool) {
		gd := entry.gd

		doc.Line("")
		if entry.doc != "" {
			doc.Line("<p>")
			doc.Line(markdown(entry.doc))
			doc.Line("</p>")
		}

		ts := gd.Specs[0].(*ast.TypeSpec)
		st := ts.Type.(*ast.StructType)
		fl := st.Fields.List

		if len(fl) == 0 {
			if header != "" {
				doc.Line("")
				doc.Line("<p>")
				doc.Line("<strong>%s</strong>: <em>none</em>", header)
				doc.Line("</p>")
				doc.Line("")
			}
			return
		}

		if header != "" {
			doc.Line("")
			doc.Line("<p>")
			doc.Line("<strong>%s</strong>: ", header)
			doc.Line("</p>")
			doc.Line("")
		}

		doc.Line("")
		doc.Line("<table class=%#v>", "field-table")
		doc.Line("<tr>")
		doc.Line("<th>Name</th>")
		doc.Line("<th>Type</th>")
		if showDesc {
			doc.Line("<th>Description</th>")
		}
		doc.Line("</tr>")
		for _, sf := range fl {
			if sf.Tag == nil {
				log.Fatalf("%s.%s is untagged", ts.Name.Name, sf.Names[0].Name)
			}

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
			doc.Line("<tr>")
			doc.Line("<td><code>%s</code></td>", jsonTag.Name)
			doc.Line("<td>%s</td>", scope.LinkType(typeToString(sf.Type), showDesc))
			if showDesc {
				doc.Line("<td>%s</td>", markdown(comment))
			}
			doc.Line("</tr>")
		}
		doc.Line("</table>")
		doc.Line("")
	}

	renderHeader := func(entry *Entry) {
		var kindString string
		switch entry.kind {
		case EntryKindParams:
			switch entry.caller {
			case CallerClient:
				kindString = `<em class="request-client-caller"></em>`
			case CallerServer:
				kindString = `<em class="request-server-caller"></em>`
			}
		case EntryKindNotification:
			kindString = `<em class="notification"></em>`
		case EntryKindType:
			kindString = `<em class="type"></em>`
		}

		doc.Line("### %s%s", kindString, entry.name)
		doc.Line("")
		doc.Line("<p class='tags'>")
		switch entry.kind {
		case EntryKindParams:
			switch entry.caller {
			case CallerClient:
				doc.Line("<em>Client request</em>")
			case CallerServer:
				doc.Line("<em>Server request</em>")
			}
		case EntryKindNotification:
			doc.Line("<em>Notification</em>")
		case EntryKindType:
			doc.Line("<em>Type</em>")
		}
		for _, tag := range entry.tags {
			doc.Line("<em>%s</em>", tag)
		}
		doc.Line("</p>")
	}

	renderRequest := func(params *Entry) {
		paramsName := asType(params.gd).Name.Name
		resultName := strings.TrimSuffix(paramsName, "Params") + "Result"

		result := scope.FindEntry(resultName)

		renderHeader(params)
		dumpStruct("Parameters", params, true)
		dumpStruct("Result", result, true)
	}

	renderNotification := func(entry *Entry) {
		renderHeader(entry)
		dumpStruct("Payload", entry, true)
	}

	renderType := func(entry *Entry) {
		renderHeader(entry)
		dumpStruct("Fields", entry, true)

		ts := entry.gd.Specs[0].(*ast.TypeSpec)

		id := ts.Name.Name + "__TypeHint"

		doc.Line("")
		doc.Line("<div id=%#v style=%#v class=%#v>", id, "display: none;", "tip-content")
		doc.Line("<p>%s <a href=%#v>(Go to definition)</a></p>", ts.Name.Name, fmt.Sprintf("#/?id=%s", linkify(ts.Name.Name)))
		dumpStruct("", entry, false)
		doc.Line("</div>")
		doc.Line("")
	}

	doc.Line("")
	doc.Line("# Messages")
	doc.Line("")

	// Make sure the Misc. category is at the end
	sort.Slice(scope.categoryList, func(i, j int) bool {
		if scope.categoryList[i] == "Miscellaneous" {
			return false
		}
		return true
	})
	for _, category := range scope.categoryList {
		doc.Line("")
		doc.Line("## %s", category)
		doc.Line("")

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

	doc.Commit("{{EVERYTHING}}")
	doc.Write()

	return nil
}

func markdown(s string) string {
	return string(blackfriday.Run([]byte(s)))
}
