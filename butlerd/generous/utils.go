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

func isStruct(ts *ast.TypeSpec) bool {
	if ts == nil {
		return false
	}

	_, ok := ts.Type.(*ast.StructType)
	return ok
}

func isEnum(ts *ast.TypeSpec) bool {
	if ts == nil {
		return false
	}

	if id, ok := ts.Type.(*ast.Ident); ok {
		if id.Name == "string" || id.Name == "int64" {
			return true
		}
	}

	return false
}

func parseTag(line string) (tag string, value string) {
	if strings.HasPrefix(line, "@") {
		for i := 1; i < len(line); i++ {
			if line[i] == ' ' {
				tag = line[1:i]
				value = line[i+1:]
				break
			}
		}
	}
	return
}

func linkify(input string) string {
	return strings.Replace(strings.ToLower(input), ".", "", -1)
}

func jsonType(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int64", "float64", "int32", "uint32":
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
		name := node.Sel.Name
		switch name {
		case "Time":
			return "Date" // javascript type date
		default:
			return name
		}
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
