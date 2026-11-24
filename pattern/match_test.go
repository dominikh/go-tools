package pattern

import (
	"go/parser"
	"testing"
)

func TestNotDoesNotBind(t *testing.T) {
	tests := []struct {
		pattern string
		expr    string
		want    bool
	}{
		// The left operand is not a sum, so Not matches. The partial match
		// of the inner pattern must not leave x bound to a.
		{`(BinaryExpr (Not (BinaryExpr x "+" _)) "*" x)`, "(a*b)*c", true},
		{`(BinaryExpr (Not (BinaryExpr _ "+" _)) "*" x)`, "(a*b)*c", true},
		{`(BinaryExpr (Not (BinaryExpr x "+" _)) "*" x)`, "(a+b)*c", false},
	}
	for _, tt := range tests {
		expr, err := parser.ParseExpr(tt.expr)
		if err != nil {
			t.Fatal(err)
		}
		pat, err := (&Parser{}).Parse(tt.pattern)
		if err != nil {
			t.Fatal(err)
		}
		m := &Matcher{}
		if got := m.Match(pat, expr); got != tt.want {
			t.Errorf("%s on %s: got %t, want %t", tt.pattern, tt.expr, got, tt.want)
		}
	}
}
