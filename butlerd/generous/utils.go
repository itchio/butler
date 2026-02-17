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
		switch id.Name {
		case "string", "int", "int32", "int64", "uint32", "uint64", "float64", "bool":
			return true
		}
	}

	return false
}

func isArrayAlias(ts *ast.TypeSpec) bool {
	if ts == nil {
		return false
	}

	if _, ok := ts.Type.(*ast.ArrayType); ok {
		return true
	}

	return false
}

func parseTag(line string) (tag string, value string) {
	if strings.HasPrefix(line, "@") {
		for i := 1; i < len(line); i++ {
			if line[i] == ' ' {
				tag = line[1:i]
				value = line[i+1:]
				return
			}
		}
		// No space found — tag with no value (e.g. "@deprecated")
		tag = line[1:]
	}
	return
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
			return "RFCDate"
		default:
			return name
		}
	case *ast.ArrayType:
		return typeToString(node.Elt) + "[]"
	case *ast.MapType:
		return "{ [key: " + typeToString(node.Key) + "]: " + typeToString(node.Value) + " }"
	case *ast.InterfaceType:
		return "any"
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
