package functions

import (
	"honnef.co/go/tools/ssa"
)

// IsStub reports whether a function is a stub. A function is
// considered a stub if it has no instructions or if all it does is
// return a constant value.
func IsStub(fn *ssa.Function) bool {
	for _, b := range fn.Blocks {
		for _, instr := range b.Instrs {
			switch instr.(type) {
			case *ssa.Const, *ssa.Panic, *ssa.Return, *ssa.DebugRef:
				// Const has no side-effects, Panic and
				// Return must be using a constant value, or there are
				// other instructions.
			default:
				return false
			}
		}
	}
	return true
}
