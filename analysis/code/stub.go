package code

import (
	"honnef.co/go/tools/analysis/facts"
	"honnef.co/go/tools/go/ir"
)

func IsStub(fn *ir.Function) bool {
	return facts.IsStub(fn)
}
