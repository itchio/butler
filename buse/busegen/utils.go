package main

import (
	"fmt"
	"go/ast"
	"strings"
)

func asType(gd *ast.GenDecl) *ast.TypeSpec {
	for _, spec := range gd.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok {
			return ts
		}
	}
	return nil
}

func linkify(input string) string {
	return strings.Replace(strings.ToLower(input), ".", "", -1)
}

func jsonType(goType string) string {
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

func typeToString(e ast.Expr) string {
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

func getCommentLines(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}

	var lines []string
	for _, el := range doc.List {
		line := strings.TrimSpace(strings.TrimPrefix(el.Text, "//"))
		lines = append(lines, line)
	}
	return lines
}

func getComment(doc *ast.CommentGroup, separator string) string {
	lines := getCommentLines(doc)
	if len(lines) == 0 {
		return "*undocumented*"
	}

	return strings.Join(lines, separator)
}
