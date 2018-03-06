package main

import (
	"fmt"
	"go/ast"
	"strings"
)

func (bc *BuseContext) GenerateDocs() error {
	bc.Task("Generating docs")

	doc := bc.NewBusegenRelativeDoc("docs/README.md")
	doc.Load(bc.ReadFile("layout.md"))

	scope := newScope(bc)
	must(scope.Assimilate("github.com/itchio/butler/buse", "types.go"))
	must(scope.Assimilate("github.com/itchio/go-itchio", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/configurator", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/installer/bfs", "receipt.go"))

	dumpStruct := func(header string, entry *Entry, showDesc bool) {
		ts := entry.typeSpec

		doc.Line("")
		if len(entry.doc) > 0 {
			doc.Line("<p>")
			doc.Line(scope.MarkdownAll(entry.doc, showDesc))
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

		for _, sf := range entry.structFields {
			var beforeDesc = ""
			if sf.optional {
				beforeDesc = fmt.Sprintf("<span class=%#v>Optional</span> ", "tag")
			}

			doc.Line("<tr>")
			doc.Line("<td><code>%s</code></td>", sf.name)
			doc.Line("<td>%s</td>", scope.LinkType(sf.typeString, showDesc))
			if showDesc {
				comment := strings.Join(sf.doc, "\n")
				doc.Line("<td>%s</td>", scope.Markdown(beforeDesc+comment, showDesc))
			}
			doc.Line("</tr>")
		}

		doc.Line("</table>")
		doc.Line("")
	}

	dumpEnum := func(header string, entry *Entry, showDesc bool) {
		doc.Line("")
		if len(entry.doc) > 0 {
			doc.Line("<p>")
			doc.Line(scope.MarkdownAll(entry.doc, showDesc))
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
	var ourCategoryList []string
	for _, category := range scope.categoryList {
		if category != "Miscellaneous" {
			ourCategoryList = append(ourCategoryList, category)
		}
	}
	ourCategoryList = append(ourCategoryList, "Miscellaneous")

	for _, category := range ourCategoryList {
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
