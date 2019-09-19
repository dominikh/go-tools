package functions

import (
	"go/types"

	"honnef.co/go/tools/ssa"
)

// Terminates reports whether fn is supposed to return, that is if it
// has at least one theoretic path that returns from the function.
// Explicit panics do not count as terminating.
func Terminates(fn *ssa.Function) bool {
	if fn.Blocks == nil {
		// assuming that a function terminates is the conservative
		// choice
		return true
	}

	for _, block := range fn.Blocks {
		if _, ok := block.Control().(*ssa.Return); ok {
			if len(block.Preds) == 0 {
				return true
			}
			for _, pred := range block.Preds {
				// Receiving from a time.Tick channel won't ever
				// return !ok, so a range loop over it won't
				// terminate.
				iff, ok := pred.Control().(*ssa.If)
				if !ok {
					return true
				}
				ex, ok := iff.Cond.(*ssa.Extract)
				if !ok {
					return true
				}
				if ex.Index != 1 {
					return true
				}
				recv, ok := ex.Tuple.(*ssa.Recv)
				if !ok {
					return true
				}
				call, ok := recv.Chan.(*ssa.Call)
				if !ok {
					return true
				}
				fn, ok := call.Common().Value.(*ssa.Function)
				if !ok {
					return true
				}
				fn2, ok := fn.Object().(*types.Func)
				if !ok {
					return true
				}
				if fn2.FullName() != "time.Tick" {
					return true
				}
			}
		}
	}
	return false
}
