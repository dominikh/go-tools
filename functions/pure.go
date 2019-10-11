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
			case *ssa.Const:
				// const naturally has no side-effects
			case *ssa.Panic:
				// panic is a stub if it only uses constants
			case *ssa.Return:
				// return is a stub if it only uses constants
			case *ssa.DebugRef:
			case *ssa.Jump:
				// if there are no disallowed instructions, then we're
				// only jumping to the exit block (or possibly
				// somewhere else that's stubby?)
			default:
				// all other instructions are assumed to do actual work
				return false
			}
		}
	}
	return true
}
