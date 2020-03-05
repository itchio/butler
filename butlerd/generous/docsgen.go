package main

import (
	"fmt"
	"go/ast"
	"regexp"
	"strings"
)

func (bc *generousContext) generateDocs() error {
	bc.task("Generating docs")

	doc := bc.newGenerousRelativeDoc("docs/README.md")
	doc.load(bc.readFile("layout.md"))

	scope := newScope(bc)
	scope.assimilateAll()

	dumpStruct := func(header string, entry *entryInfo, showDesc bool) {
		ts := entry.typeSpec

		doc.line("")
		if len(entry.doc) > 0 {
			doc.line("<p>")
			doc.line(scope.markdownAll(entry.doc, showDesc))
			doc.line("</p>")
		}

		st := ts.Type.(*ast.StructType)
		fl := st.Fields.List

		if len(fl) == 0 {
			if header != "" {
				doc.line("")
				doc.line("<p>")
				doc.line("<span class=%#v>%s</span> <em>none</em>", "header", header)
				doc.line("</p>")
				doc.line("")
			}
			return
		}

		if header != "" {
			doc.line("")
			doc.line("<p>")
			doc.line("<span class=%#v>%s</span> ", "header", header)
			doc.line("</p>")
			doc.line("")
		}

		doc.line("")
		doc.line("<table class=%#v>", "field-table")

		for _, sf := range entry.structFields {
			var beforeDesc = ""
			if sf.optional {
				beforeDesc = fmt.Sprintf("<span class=%#v>Optional</span> ", "tag")
			}

			doc.line("<tr>")
			doc.line("<td><code>%s</code></td>", sf.name)
			doc.line("<td>%s</td>", scope.linkType(sf.typeString, showDesc))
			if showDesc {
				comment := strings.Join(sf.doc, "\n")
				doc.line("<td>%s</td>", scope.markdown(beforeDesc+comment, showDesc))
			}
			doc.line("</tr>")
		}

		doc.line("</table>")
		doc.line("")
	}

	dumpEnum := func(header string, entry *entryInfo, showDesc bool) {
		doc.line("")
		if len(entry.doc) > 0 {
			doc.line("<p>")
			doc.line(scope.markdownAll(entry.doc, showDesc))
			doc.line("</p>")
		}

		values := entry.enumValues

		if len(values) == 0 {
			if header != "" {
				doc.line("")
				doc.line("<p>")
				doc.line("<span class=%#v>%s</span> <em>none</em>", "header", header)
				doc.line("</p>")
				doc.line("")
			}
			return
		}

		if header != "" {
			doc.line("")
			doc.line("<p>")
			doc.line("<span class=%#v>%s</span> ", "header", header)
			doc.line("</p>")
			doc.line("")
		}

		doc.line("")
		doc.line("<table class=%#v>", "field-table")

		for _, v := range values {
			doc.line("<tr>")
			doc.line("<td><code>%s</code></td>", v.value)
			if showDesc {
				comment := strings.Join(v.doc, "\n")
				doc.line("<td>%s</td>", scope.markdown(comment, showDesc))
			}
			doc.line("</tr>")
		}
		doc.line("</table>")
		doc.line("")
	}

	kindString := func(entry *entryInfo) string {
		switch entry.kind {
		case entryKindParams:
			switch entry.caller {
			case callerClient:
				return `(client request)`
			case callerServer:
				return `(client caller)`
			}
		case entryKindNotification:
			return `(notification)`
		case entryKindType:
			return `(struct)`
		case entryKindEnum:
			return `(enum)`
		}
		return ""
	}

	renderHeader := func(entry *entryInfo) {
		doc.line("### %s %s", entry.name, kindString(entry))
		doc.line("")
	}

	invalidHrefCharacters := regexp.MustCompile("[^A-Za-z-]")
	getHref := func(entry *entryInfo) string {
		id := fmt.Sprintf("%s %s", entry.name, kindString(entry))
		id = strings.Replace(id, " ", "-", -1)
		id = invalidHrefCharacters.ReplaceAllString(id, "")
		id = strings.ToLower(id)
		return fmt.Sprintf("#/?id=%s", id)
	}

	renderTypeHint := func(entry *entryInfo) {
		id := entry.typeName + "__TypeHint"
		doc.line("")
		doc.line("<div id=%#v class=%#v>", id, "tip-content")
		doc.line("<p>%s %s <a href=%#v>(Go to definition)</a></p>", entry.name, kindString(entry), getHref(entry))

		switch entry.typeKind {
		case entryTypeKindStruct:
			dumpStruct("", entry, false)
		case entryTypeKindEnum:
			dumpEnum("", entry, false)
		}
		doc.line("</div>")
		doc.line("")
	}

	renderRequest := func(params *entryInfo) {
		paramsName := asType(params.gd).Name.Name
		resultName := strings.TrimSuffix(paramsName, "Params") + "Result"

		result := scope.findEntry(resultName)

		renderHeader(params)
		dumpStruct("Parameters", params, true)
		dumpStruct("Result", result, true)

		renderTypeHint(params)
		renderTypeHint(result)
	}

	renderNotification := func(entry *entryInfo) {
		renderHeader(entry)
		dumpStruct("Payload", entry, true)

		renderTypeHint(entry)
	}

	renderType := func(entry *entryInfo) {
		renderHeader(entry)
		dumpStruct("Fields", entry, true)

		renderTypeHint(entry)
	}

	renderEnum := func(entry *entryInfo) {
		renderHeader(entry)
		dumpEnum("Values", entry, true)

		renderTypeHint(entry)
	}

	renderAlias := func(entry *entryInfo) {
		renderHeader(entry)
		doc.line("")
		if len(entry.doc) > 0 {
			doc.line("<p>")
			doc.line(scope.markdownAll(entry.doc, true))
			doc.line("</p>")
		}
		if id, ok := entry.typeSpec.Type.(*ast.Ident); ok {
			doc.line("Type alias for %s", id.Name)
		} else {
			doc.line("Type alias")
		}

		renderTypeHint(entry)
	}

	doc.line("")
	doc.line("# Messages")
	doc.line("")

	// Make sure the Misc. category is at the end
	var ourCategoryList []string
	for _, category := range scope.categoryList {
		if category != "Miscellaneous" {
			ourCategoryList = append(ourCategoryList, category)
		}
	}
	ourCategoryList = append(ourCategoryList, "Miscellaneous")

	for _, category := range ourCategoryList {
		doc.line("")
		doc.line("## %s Category", category)
		doc.line("")

		cat := scope.categories[category]
		for _, entry := range cat.entries {
			switch entry.kind {
			case entryKindParams:
				renderRequest(entry)
			case entryKindNotification:
				renderNotification(entry)
			case entryKindType:
				renderType(entry)
			case entryKindEnum:
				renderEnum(entry)
			case entryKindAlias:
				renderAlias(entry)
			}
		}
	}

	doc.commit("{{EVERYTHING}}")
	doc.write()

	return nil
}
