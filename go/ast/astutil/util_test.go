package astutil

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestCopyExprCopiesIndexListIndices(t *testing.T) {
	expr, err := parser.ParseExpr("f[int, string]")
	if err != nil {
		t.Fatal(err)
	}
	original := expr.(*ast.IndexListExpr)
	copy, ok := CopyExpr(original)
	if !ok {
		t.Fatal("CopyExpr failed")
	}
	copied := copy.(*ast.IndexListExpr)
	if &original.Indices[0] == &copied.Indices[0] {
		t.Fatal("CopyExpr shares the index slice")
	}
	copied.Indices[0] = ast.NewIdent("bool")
	if original.Indices[0].(*ast.Ident).Name != "int" {
		t.Fatal("modifying the copy changed the original")
	}
}
