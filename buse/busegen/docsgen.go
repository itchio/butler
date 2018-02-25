package main

import (
	"fmt"
	"go/ast"
	"log"
	"sort"
	"strings"

	"github.com/fatih/structtag"
)

func (bc *BuseContext) GenerateDocs() error {
	bc.Task("Generating docs")

	doc := bc.NewDoc("docs/README.md")
	doc.Load(bc.ReadFile("layout.md"))

	scope := newScope()
	must(scope.Assimilate("github.com/itchio/butler/buse", "types.go"))
	must(scope.Assimilate("github.com/itchio/go-itchio", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/configurator", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/installer/bfs", "receipt.go"))
	must(scope.Assimilate("github.com/itchio/butler/endpoints/launch/manifest", "types.go"))

	dumpStruct := func(header string, entry *Entry, showDesc bool) {
		ts := entry.typeSpec

		doc.Line("")
		if entry.doc != "" {
			doc.Line("<p>")
			doc.Line(scope.Markdown(entry.doc, showDesc))
			doc.Line("</p>")
		}

		st := ts.Type.(*ast.StructType)
		fl := st.Fields.List

		if len(fl) == 0 {
			if header != "" {
				doc.Line("")
				doc.Line("<p>")
				doc.Line("<span class=%#v>%s</span> <em>none</em>", "header", header)
				doc.Line("</p>")
				doc.Line("")
			}
			return
		}

		if header != "" {
			doc.Line("")
			doc.Line("<p>")
			doc.Line("<span class=%#v>%s</span> ", "header", header)
			doc.Line("</p>")
			doc.Line("")
		}

		doc.Line("")
		doc.Line("<table class=%#v>", "field-table")

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

			var beforeDesc = ""
			comment := getComment(sf.Doc, "\n")
			if strings.Contains(comment, "@optional") {
				comment = strings.TrimSpace(strings.Replace(comment, "@optional", "", -1))
				beforeDesc = fmt.Sprintf("<span class=%#v>Optional</span> ", "tag")
			}

			doc.Line("<tr>")
			doc.Line("<td><code>%s</code></td>", jsonTag.Name)
			doc.Line("<td>%s</td>", scope.LinkType(typeToString(sf.Type), showDesc))
			if showDesc {
				doc.Line("<td>%s</td>", scope.Markdown(beforeDesc+comment, showDesc))
			}
			doc.Line("</tr>")
		}
		doc.Line("</table>")
		doc.Line("")
	}

	dumpEnum := func(header string, entry *Entry, showDesc bool) {
		doc.Line("")
		if entry.doc != "" {
			doc.Line("<p>")
			doc.Line(scope.Markdown(entry.doc, showDesc))
			doc.Line("</p>")
		}

		values := entry.enumValues

		if len(values) == 0 {
			if header != "" {
				doc.Line("")
				doc.Line("<p>")
				doc.Line("<span class=%#v>%s</span> <em>none</em>", "header", header)
				doc.Line("</p>")
				doc.Line("")
			}
			return
		}

		if header != "" {
			doc.Line("")
			doc.Line("<p>")
			doc.Line("<span class=%#v>%s</span> ", "header", header)
			doc.Line("</p>")
			doc.Line("")
		}

		doc.Line("")
		doc.Line("<table class=%#v>", "field-table")

		for _, v := range values {
			doc.Line("<tr>")
			doc.Line("<td><code>%s</code></td>", v.value)
			if showDesc {
				comment := strings.Join(v.doc, "\n")
				doc.Line("<td>%s</td>", scope.Markdown(comment, showDesc))
			}
			doc.Line("</tr>")
		}
		doc.Line("</table>")
		doc.Line("")
	}

	kindString := func(entry *Entry) string {
		switch entry.kind {
		case EntryKindParams:
			switch entry.caller {
			case CallerClient:
				return `<em class="request-client-caller"></em>`
			case CallerServer:
				return `<em class="request-server-caller"></em>`
			}
		case EntryKindNotification:
			return `<em class="notification"></em>`
		case EntryKindType:
			return `<em class="struct-type"></em>`
		case EntryKindEnum:
			return `<em class="enum-type"></em>`
		}
		return ""
	}

	renderHeader := func(entry *Entry) {
		doc.Line("### %s%s", kindString(entry), entry.name)
		doc.Line("")
	}

	renderTypeHint := func(entry *Entry) {
		id := entry.typeName + "__TypeHint"
		doc.Line("")
		doc.Line("<div id=%#v style=%#v class=%#v>", id, "display: none;", "tip-content")
		doc.Line("<p>%s%s <a href=%#v>(Go to definition)</a></p>", kindString(entry), entry.name, fmt.Sprintf("#/?id=%s", linkify(entry.name)))

		switch entry.typeKind {
		case EntryTypeKindStruct:
			dumpStruct("", entry, false)
		case EntryTypeKindEnum:
			dumpEnum("", entry, false)
		}
		doc.Line("</div>")
		doc.Line("")
	}

	renderRequest := func(params *Entry) {
		paramsName := asType(params.gd).Name.Name
		resultName := strings.TrimSuffix(paramsName, "Params") + "Result"

		result := scope.FindEntry(resultName)

		renderHeader(params)
		dumpStruct("Parameters", params, true)
		dumpStruct("Result", result, true)

		renderTypeHint(params)
	}

	renderNotification := func(entry *Entry) {
		renderHeader(entry)
		dumpStruct("Payload", entry, true)

		renderTypeHint(entry)
	}

	renderType := func(entry *Entry) {
		renderHeader(entry)
		dumpStruct("Fields", entry, true)

		renderTypeHint(entry)
	}

	renderEnum := func(entry *Entry) {
		renderHeader(entry)
		dumpEnum("Values", entry, true)

		renderTypeHint(entry)
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
			case EntryKindEnum:
				renderEnum(entry)
			}
		}
	}

	doc.Commit("{{EVERYTHING}}")
	doc.Write()

	return nil
}
