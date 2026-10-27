package astutil

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestEqualCompositeLit(t *testing.T) {
	parse := func(src string) ast.Expr {
		expr, err := parser.ParseExpr(src)
		if err != nil {
			t.Fatal(err)
		}
		return expr
	}

	if Equal(parse("T{1}"), parse("T{2}")) {
		t.Fatal("different composite literals are equal")
	}
	if !Equal(parse("T{1}"), parse("T{1}")) {
		t.Fatal("identical composite literals are not equal")
	}
}
