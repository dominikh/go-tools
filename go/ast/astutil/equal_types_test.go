package astutil

import (
	"go/ast"
	"go/parser"
	"testing"
)

func parseExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	expr, err := parser.ParseExpr(src)
	if err != nil {
		t.Fatal(err)
	}
	return expr
}

func TestEqualChannelElementType(t *testing.T) {
	if Equal(parseExpr(t, "chan int"), parseExpr(t, "chan string")) {
		t.Fatal("Equal considered channels with different element types equal")
	}
}

func TestEqualFunctionType(t *testing.T) {
	if Equal(parseExpr(t, "func(int)"), parseExpr(t, "func(string)")) {
		t.Fatal("Equal considered function types with different parameter types equal")
	}
	if Equal(parseExpr(t, "func()"), parseExpr(t, "func() int")) {
		t.Fatal("Equal considered function types with different result types equal")
	}
}
