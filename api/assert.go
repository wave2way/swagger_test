package api

import (
	"bytes"
	"fmt"
	"go/ast"
)

// ExprString returns the (possibly shortened) string representation for x.
// Shortened representations are suitable for user interfaces but may not
// necessarily follow Go syntax.
func ExprString(x ast.Expr) string {
	var buf bytes.Buffer
	WriteExpr(&buf, x)
	return buf.String()
}

// WriteExpr writes the (possibly shortened) string representation for x to buf.
// Shortened representations are suitable for user interfaces but may not
// necessarily follow Go syntax.
func WriteExpr(buf *bytes.Buffer, expr ast.Expr) {
	// The AST preserves source-level parentheses so there is
	// no need to introduce them here to correct for different
	// operator precedences. (This assumes that the AST was
	// generated by a Go parser.)

	switch x := expr.(type) {
	default:
		buf.WriteString(fmt.Sprintf("(ast: %T)", x)) // nil, ast.BadExpr, ast.KeyValueExpr

	case *ast.Ident:
		buf.WriteString(x.Name)

	case *ast.Ellipsis:
		buf.WriteString("[]")
		if x.Elt != nil {
			WriteExpr(buf, x.Elt)
		}

	case *ast.BasicLit:
		buf.WriteString(x.Value)

	case *ast.FuncLit:
		buf.WriteByte('(')
		WriteExpr(buf, x.Type)
		buf.WriteString(" literal)") // shortened

	case *ast.CompositeLit:
		buf.WriteByte('(')
		WriteExpr(buf, x.Type)
		buf.WriteString(" literal)") // shortened

	case *ast.ParenExpr:
		buf.WriteByte('(')
		WriteExpr(buf, x.X)
		buf.WriteByte(')')

	case *ast.SelectorExpr:
		WriteExpr(buf, x.X)
		buf.WriteByte('.')
		buf.WriteString(x.Sel.Name)

	// case *ast.IndexExpr:
	// 	WriteExpr(buf, x.X)
	// 	buf.WriteByte('[')
	// 	exprs := typeparams.UnpackExpr(x.Index)
	// 	for i, e := range exprs {
	// 		if i > 0 {
	// 			buf.WriteString(", ")
	// 		}
	// 		WriteExpr(buf, e)
	// 	}
	// 	buf.WriteByte(']')

	case *ast.SliceExpr:
		buf.WriteString("[]")
		WriteExpr(buf, x.X)

	case *ast.TypeAssertExpr:
		WriteExpr(buf, x.Type)

	case *ast.CallExpr:
		WriteExpr(buf, x.Fun)
		buf.WriteByte('(')
		writeExprList(buf, x.Args)
		if x.Ellipsis.IsValid() {
			buf.WriteString("[]")
		}
		buf.WriteByte(')')

	case *ast.StarExpr:
		buf.WriteByte('*')
		WriteExpr(buf, x.X)

	case *ast.UnaryExpr:
		WriteExpr(buf, x.X)

	case *ast.BinaryExpr:
		WriteExpr(buf, x.X)
		buf.WriteByte(' ')
		buf.WriteByte(' ')
		WriteExpr(buf, x.Y)

	case *ast.ArrayType:
		buf.WriteByte('[')
		if x.Len != nil {
			WriteExpr(buf, x.Len)
		}
		buf.WriteByte(']')
		WriteExpr(buf, x.Elt)

	case *ast.StructType:
		WriteExpr(buf, expr)
		// buf.WriteString("struct{")
		// writeFieldList(buf, x.Fields.List, "; ", false)
		// buf.WriteByte('}')

	case *ast.FuncType:
		buf.WriteString("func")
		writeSigExpr(buf, x)

	case *ast.InterfaceType:
		// separate type list types from method list
		// TODO(gri) we can get rid of this extra code if writeExprList does the separation
		var types []ast.Expr
		var methods []*ast.Field
		for _, f := range x.Methods.List {
			if len(f.Names) > 1 && f.Names[0].Name == "type" {
				// type list type
				types = append(types, f.Type)
			} else {
				// method or embedded interface
				methods = append(methods, f)
			}
		}

		buf.WriteString("interface{")
		writeFieldList(buf, methods, "; ", true)
		if len(types) > 0 {
			if len(methods) > 0 {
				buf.WriteString("; ")
			}
			buf.WriteString("type ")
			writeExprList(buf, types)
		}
		buf.WriteByte('}')

	case *ast.MapType:
		buf.WriteString("map[")
		WriteExpr(buf, x.Key)
		buf.WriteByte(']')
		WriteExpr(buf, x.Value)

	case *ast.ChanType:
		var s string
		switch x.Dir {
		case ast.SEND:
			s = "chan<- "
		case ast.RECV:
			s = "<-chan "
		default:
			s = "chan "
		}
		buf.WriteString(s)
		WriteExpr(buf, x.Value)
	}
}

func writeSigExpr(buf *bytes.Buffer, sig *ast.FuncType) {
	buf.WriteByte('(')
	writeFieldList(buf, sig.Params.List, ", ", false)
	buf.WriteByte(')')

	res := sig.Results
	n := res.NumFields()
	if n == 0 {
		// no result
		return
	}

	buf.WriteByte(' ')
	if n == 1 && len(res.List[0].Names) == 0 {
		// single unnamed result
		WriteExpr(buf, res.List[0].Type)
		return
	}

	// multiple or named result(s)
	buf.WriteByte('(')
	writeFieldList(buf, res.List, ", ", false)
	buf.WriteByte(')')
}

func writeFieldList(buf *bytes.Buffer, list []*ast.Field, sep string, iface bool) {
	for i, f := range list {
		if i > 0 {
			buf.WriteString(sep)
		}

		// field list names
		// writeIdentList(buf, f.Names)

		// types of interface methods consist of signatures only
		if sig, _ := f.Type.(*ast.FuncType); sig != nil && iface {
			writeSigExpr(buf, sig)
			continue
		}

		// named fields are separated with a blank from the field type
		if len(f.Names) > 0 {
			buf.WriteByte(' ')
		}

		WriteExpr(buf, f.Type)

		// ignore tag
	}
}

func writeIdentList(buf *bytes.Buffer, list []*ast.Ident) {
	for i, x := range list {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(x.Name)
	}
}

func writeExprList(buf *bytes.Buffer, list []ast.Expr) {
	for i, x := range list {
		if i > 0 {
			buf.WriteString(", ")
		}
		WriteExpr(buf, x)
	}
}
